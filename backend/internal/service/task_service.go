package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"ml-platform/internal/executor"
	"ml-platform/internal/model"
	"ml-platform/internal/repository"
	"ml-platform/pkg/cosclient"
)

// cosRegion 从环境变量读取，用于 coscli 命令的 -r 参数
var cosRegion = os.Getenv("TENCENT_CLOUD_COS_REGION")

// TaskService 任务服务
type TaskService struct {
	repo            *repository.TaskRepository
	executor        *executor.Executor
	storage         *StorageService
	instanceService *InstanceService
	tasks           map[string]*model.Task // 本地缓存，用于快速访问
	mu              sync.RWMutex
}

// NewTaskService 创建任务服务
func NewTaskService(repo *repository.TaskRepository, exec *executor.Executor, cosClient *cosclient.Client, instanceService *InstanceService) *TaskService {
	return &TaskService{
		repo:            repo,
		executor:        exec,
		storage:         NewStorageService(cosClient, nil),
		instanceService: instanceService,
		tasks:           make(map[string]*model.Task),
	}
}

// CreateTask 创建任务
func (s *TaskService) CreateTask(ctx context.Context, req *model.CreateTaskRequest) (*model.Task, error) {
	task := model.NewTask(req.Name, req.Image, req.Command)
	task.Description = req.Description
	task.DataPath = req.DataPath
	task.OutputPath = req.OutputPath
	task.EnvVars = model.StringSlice(req.EnvVars)
	task.StorageBindings = model.StorageBindingList(req.StorageBindings)

	// 如果指定了 Provider，创建远程实例
	if req.Provider != "" {
		task.Provider = req.Provider
		task.Phase = model.TaskPhaseCreatingInstance

		// 先保存任务（获取ID）
		if err := s.repo.Create(ctx, task); err != nil {
			return nil, fmt.Errorf("failed to create task in database: %w", err)
		}

		// 创建实例
		instance, err := s.instanceService.CreateOffer(ctx, &model.CreateOfferRequest{
			Provider:      req.Provider,
			OfferID:       req.OfferID,
			UserID:        "default",
			Name:          fmt.Sprintf("task-%s", task.ID),
			Image:         req.Image,
			DiskSize:      req.DiskSize,
			DurationHours: req.DurationHours,
			PricePerHour:  req.PricePerHour,
			GPUName:       req.GPUName,
			NumGPUs:       req.NumGPUs,
		})
		if err != nil {
			task.Status = model.TaskStatusFailed
			task.Phase = model.TaskPhaseFailed
			task.ErrorMsg = fmt.Sprintf("failed to create instance: %v", err)
			s.repo.Update(ctx, task)
			return task, nil // 返回任务，但状态为失败
		}

		task.InstanceID = instance.ID
		// 保存 instance 关联
		if err := s.repo.Update(ctx, task); err != nil {
			log.Printf("Failed to update task with instance ID: %v", err)
		}
	}

	// 未指定 Provider，直接本地执行
	if task.InstanceID == "" {
		if err := s.repo.Create(ctx, task); err != nil {
			return nil, fmt.Errorf("failed to create task in database: %w", err)
		}
	}

	// 更新本地缓存
	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	// 异步执行任务
	go s.executeTask(task)

	return task, nil
}

// executeTask 异步执行任务
func (s *TaskService) executeTask(task *model.Task) {
	ctx := context.Background()

	if task.InstanceID != "" {
		// 远程实例执行
		s.executeRemoteTask(ctx, task)
		return
	}

	// 本地 Docker 执行（保持原有行为）
	task.Status = model.TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now

	if err := s.repo.Update(ctx, task); err != nil {
		log.Printf("Failed to update task status: %v", err)
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	// 执行本地任务
	if err := s.executor.ExecuteTask(ctx, task); err != nil {
		task.Status = model.TaskStatusFailed
		task.ErrorMsg = err.Error()
		log.Printf("Task %s execution error: %v", task.ID, err)
	} else {
		task.Status = model.TaskStatusCompleted
		log.Printf("Task %s completed successfully", task.ID)
	}

	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if err := s.repo.Update(ctx, task); err != nil {
		log.Printf("Failed to update task completion: %v", err)
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	// 如果有输出路径，上传到COS
	if task.OutputPath != "" && task.Status == model.TaskStatusCompleted {
		outputDir := fmt.Sprintf("/app/output/%s", task.ID)
		if err := s.storage.UploadResults(ctx, task.ID, task.OutputPath, outputDir); err != nil {
			log.Printf("Failed to upload results for task %s: %v", task.ID, err)
		}
	}
}

// executeRemoteTask 在远程实例上执行训练任务
func (s *TaskService) executeRemoteTask(ctx context.Context, task *model.Task) {
	log.Printf("[task %s] Starting remote execution on instance %s", task.ID, task.InstanceID)

	// 1. 等待实例就绪并获取 SSH 信息
	inst, sshClient, err := s.waitForInstance(ctx, task)
	if err != nil {
		// Dummy provider 没有真实 SSH 服务，回退到本地执行
		if task.Provider == "local" || task.Provider == "dummy" {
			s.emitLog(task.ID, "info", "Dummy provider detected, falling back to local execution")
			task.InstanceID = ""
			task.Phase = model.TaskPhasePending
			s.executeTask(task)
			return
		}
		s.failTask(ctx, task, fmt.Sprintf("instance not ready: %v", err))
		return
	}
	defer sshClient.Close()

	// 保存 SSH 信息到任务（用于后续日志获取等）
	task.SSHHost = inst.SSHHost
	task.SSHPort = inst.SSHPort
	task.SSHUser = inst.SSHUser
	task.SSHPassword = inst.Password

	// 2. 创建工作目录
	s.emitLog(task.ID, "info", "Preparing workspace on remote instance...")
	if len(task.StorageBindings) > 0 {
		for _, binding := range task.StorageBindings {
			dir := fmt.Sprintf("/app/bindings/%s", binding.EnvName)
			if _, stderr, err := sshClient.Exec(fmt.Sprintf("mkdir -p \"%s\"", dir)); err != nil {
				s.failTask(ctx, task, fmt.Sprintf("failed to create workspace %s: %v, stderr: %s", dir, err, stderr))
				return
			}
		}
	} else {
		if _, stderr, err := sshClient.Exec("mkdir -p /app/data /app/output"); err != nil {
			s.failTask(ctx, task, fmt.Sprintf("failed to create workspace: %v, stderr: %s", err, stderr))
			return
		}
	}

	// 3. 下载输入数据
	if len(task.StorageBindings) > 0 {
		var inputBindings []model.StorageBinding
		for _, b := range task.StorageBindings {
			if b.Type == "input" && b.Path != "" {
				inputBindings = append(inputBindings, b)
			}
		}
		if len(inputBindings) > 0 {
			task.Phase = model.TaskPhaseDownloading
			s.updateTask(ctx, task)
			for _, binding := range inputBindings {
				localDir := fmt.Sprintf("/app/bindings/%s", binding.EnvName)
				if err := s.downloadBindingData(ctx, task, sshClient, binding.Path, localDir); err != nil {
					s.failTask(ctx, task, fmt.Sprintf("failed to download input %s: %v", binding.EnvName, err))
					return
				}
			}
		}
	} else if task.DataPath != "" {
		task.Phase = model.TaskPhaseDownloading
		s.updateTask(ctx, task)
		if err := s.downloadInputData(ctx, task, sshClient); err != nil {
			s.failTask(ctx, task, fmt.Sprintf("failed to download input: %v", err))
			return
		}
	}

	// 4. 启动训练
	task.Status = model.TaskStatusRunning
	task.Phase = model.TaskPhaseTraining
	now := time.Now()
	task.StartedAt = &now
	s.updateTask(ctx, task)

	var exitCode int
	var containerName string

	if task.Provider == "ppio" {
		// PPIO 路径：SSH 进去后直接执行训练命令（PPIO 实例本身就是容器）
		pid, err := s.startPPIOTraining(ctx, task, sshClient)
		if err != nil {
			s.failTask(ctx, task, fmt.Sprintf("failed to start training: %v", err))
			return
		}
		s.emitLog(task.ID, "info", "Training started, monitoring process...")
		exitCode = s.monitorPPIOTraining(ctx, task, sshClient, pid)
	} else {
		// Docker 路径：在远程实例上启动 Docker 容器
		containerName = fmt.Sprintf("task-%s", task.ID)
		if err := s.startTrainingContainer(ctx, task, sshClient, containerName); err != nil {
			s.failTask(ctx, task, fmt.Sprintf("failed to start training container: %v", err))
			return
		}
		s.emitLog(task.ID, "info", "Training started, monitoring container...")
		exitCode = s.monitorTraining(ctx, task, sshClient, containerName)
	}

	// 5. 训练结束，处理结果
	if exitCode == 0 {
		task.Status = model.TaskStatusCompleted
		task.Phase = model.TaskPhaseUploading
		s.emitLog(task.ID, "info", "Training completed successfully")
	} else if task.Status == model.TaskStatusCancelled {
		s.emitLog(task.ID, "info", "Training cancelled")
	} else {
		task.Status = model.TaskStatusFailed
		task.Phase = model.TaskPhaseFailed
		task.ErrorMsg = fmt.Sprintf("Training process exited with code %d", exitCode)
		s.emitLog(task.ID, "error", task.ErrorMsg)
	}
	task.ExitCode = exitCode
	s.updateTask(ctx, task)

	// 6. 上传输出数据
	if task.Status == model.TaskStatusCompleted {
		if len(task.StorageBindings) > 0 {
			for _, binding := range task.StorageBindings {
				if binding.Type == "output" && binding.Path != "" {
					localDir := fmt.Sprintf("/app/bindings/%s", binding.EnvName)
					if err := s.uploadBindingData(ctx, task, sshClient, binding.Path, localDir); err != nil {
						log.Printf("[task %s] Failed to upload output %s: %v", task.ID, binding.EnvName, err)
						s.emitLog(task.ID, "warn", fmt.Sprintf("Output upload %s failed: %v", binding.EnvName, err))
					} else {
						s.emitLog(task.ID, "info", fmt.Sprintf("Output %s uploaded to object storage", binding.EnvName))
					}
				}
			}
		} else if task.OutputPath != "" {
			if err := s.uploadOutputData(ctx, task, sshClient); err != nil {
				log.Printf("[task %s] Failed to upload output: %v", task.ID, err)
				s.emitLog(task.ID, "warn", fmt.Sprintf("Output upload failed: %v", err))
			} else {
				s.emitLog(task.ID, "info", "Output uploaded to object storage")
			}
		}
	}

	// 7. 清理并销毁实例
	if task.Provider != "ppio" && containerName != "" {
		sshClient.Exec(fmt.Sprintf("docker rm -f %s 2>/dev/null", containerName))
	}

	completedAt := time.Now()
	task.CompletedAt = &completedAt
	if task.Status == model.TaskStatusCompleted {
		task.Phase = model.TaskPhaseCompleted
	}
	s.updateTask(ctx, task)

	// 销毁实例
	// if err := s.instanceService.DestroyInstance(ctx, task.InstanceID); err != nil {
	// 	log.Printf("[task %s] Failed to destroy instance: %v", task.ID, err)
	// }

	log.Printf("[task %s] Remote execution finished, status=%s", task.ID, task.Status)
}

// waitForInstance 等待实例就绪，返回实例信息和 SSH 客户端
func (s *TaskService) waitForInstance(ctx context.Context, task *model.Task) (*model.Instance, *executor.SSHClient, error) {
	maxRetries := 60 // 最多等 5 分钟（每 5 秒一次）
	for i := 0; i < maxRetries; i++ {
		inst, err := s.instanceService.GetInstance(ctx, task.InstanceID)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		if inst.Status == model.InstanceStatusRunning && inst.SSHHost != "" && inst.SSHPort > 0 {
			// 尝试 SSH 连接
			log.Printf("[task %s] instance SSH info from DB: host=%s, port=%d, user=%s", task.ID, inst.SSHHost, inst.SSHPort, inst.SSHUser)
			sshClient := executor.NewSSHClient(inst.SSHHost, inst.SSHPort, inst.SSHUser, inst.Password)
			if err := sshClient.Connect(); err != nil {
				log.Printf("[task %s] SSH connect attempt %d failed: %v", task.ID, i+1, err)
				time.Sleep(5 * time.Second)
				continue
			}
			task.Phase = model.TaskPhaseInstanceReady
			s.updateTask(ctx, task)
			s.emitLog(task.ID, "info", fmt.Sprintf("Instance ready: %s:%d", inst.SSHHost, inst.SSHPort))
			return inst, sshClient, nil
		}

		time.Sleep(5 * time.Second)
	}
	return nil, nil, fmt.Errorf("instance not ready after %d retries", maxRetries)
}

// downloadInputData 通过 SSH 在实例上下载输入数据
func (s *TaskService) downloadInputData(ctx context.Context, task *model.Task, sshClient *executor.SSHClient) error {
	// 解析 DataPath: bucket/prefix
	parts := strings.SplitN(task.DataPath, "/", 2)
	if len(parts) < 1 {
		return fmt.Errorf("invalid data path format: %s", task.DataPath)
	}
	bucketName := parts[0]
	prefix := ""
	if len(parts) > 1 {
		prefix = parts[1]
	}

	s.emitLog(task.ID, "info", fmt.Sprintf("Downloading input data from %s/%s...", bucketName, prefix))

	// 生成预签名下载 URL
	downloads, err := s.storage.GenerateDownloadURLs(ctx, bucketName, prefix)
	if err != nil {
		return fmt.Errorf("failed to generate download URLs: %w", err)
	}

	if len(downloads) == 0 {
		s.emitLog(task.ID, "warn", "No input files found")
		return nil
	}

	// 通过 SSH 在实例上用 curl 下载每个文件
	for _, dl := range downloads {
		// 去掉 bucket 前缀，得到相对路径
		localPath := "/app/data/" + dl.Key
		// 创建子目录
		dirCmd := fmt.Sprintf("mkdir -p \"%s\"", dirOf(localPath))
		sshClient.Exec(dirCmd)

		cmd := fmt.Sprintf("coscli cp -f %s %s", shellQuote(fmt.Sprintf("cos://%s/%s", bucketName, dl.Key)), shellQuote(localPath))
		if cosRegion != "" {
			cmd += fmt.Sprintf(" -r %s", shellQuote(cosRegion))
		}
		log.Printf("[task %s] SSH download command: %s", task.ID, cmd)
		if _, stderr, err := sshClient.Exec(cmd); err != nil {
			s.emitLog(task.ID, "warn", fmt.Sprintf("Failed to download %s: %v", dl.Key, err))
			if stderr != "" {
				log.Printf("[task %s] download stderr: %s", task.ID, stderr)
			}
		} else {
			s.emitLog(task.ID, "info", fmt.Sprintf("Downloaded %s", dl.Key))
		}
	}

	return nil
}

// downloadBindingData 通过 SSH 在实例上下载指定 COS 路径到本地目录
func (s *TaskService) downloadBindingData(ctx context.Context, task *model.Task, sshClient *executor.SSHClient, cosPath string, localBaseDir string) error {
	parts := strings.SplitN(cosPath, "/", 2)
	if len(parts) < 1 {
		return fmt.Errorf("invalid path format: %s", cosPath)
	}
	bucketName := parts[0]
	prefix := ""
	if len(parts) > 1 {
		prefix = parts[1]
	}

	s.emitLog(task.ID, "info", fmt.Sprintf("Downloading from %s/%s to %s...", bucketName, prefix, localBaseDir))

	downloads, err := s.storage.GenerateDownloadURLs(ctx, bucketName, prefix)
	if err != nil {
		return fmt.Errorf("failed to generate download URLs: %w", err)
	}

	if len(downloads) == 0 {
		s.emitLog(task.ID, "warn", fmt.Sprintf("No files found in %s/%s", bucketName, prefix))
		return nil
	}

	for _, dl := range downloads {
		localPath := localBaseDir + "/" + dl.Key
		dirCmd := fmt.Sprintf("mkdir -p \"%s\"", dirOf(localPath))
		sshClient.Exec(dirCmd)

		cmd := fmt.Sprintf("coscli cp -f %s %s", shellQuote(fmt.Sprintf("cos://%s/%s", bucketName, dl.Key)), shellQuote(localPath))
		if cosRegion != "" {
			cmd += fmt.Sprintf(" -r %s", shellQuote(cosRegion))
		}
		log.Printf("[task %s] SSH binding download command: %s", task.ID, cmd)
		if _, stderr, err := sshClient.Exec(cmd); err != nil {
			s.emitLog(task.ID, "warn", fmt.Sprintf("Failed to download %s: %v", dl.Key, err))
			if stderr != "" {
				log.Printf("[task %s] download stderr: %s", task.ID, stderr)
			}
		} else {
			s.emitLog(task.ID, "info", fmt.Sprintf("Downloaded %s", dl.Key))
		}
	}

	return nil
}

// startTrainingContainer 在远程实例上启动 Docker 训练容器
func (s *TaskService) startTrainingContainer(ctx context.Context, task *model.Task, sshClient *executor.SSHClient, containerName string) error {
	// 构建环境变量和卷挂载
	envVars := []string{fmt.Sprintf("-e TASK_ID=%s", task.ID)}
	volumes := []string{}

	if len(task.StorageBindings) > 0 {
		for _, binding := range task.StorageBindings {
			localDir := fmt.Sprintf("/app/bindings/%s", binding.EnvName)
			containerDir := fmt.Sprintf("/bindings/%s", binding.EnvName)
			volumes = append(volumes, fmt.Sprintf("-v %s:%s", localDir, containerDir))
			envVars = append(envVars, fmt.Sprintf("-e %s=%s", binding.EnvName, containerDir))
		}
	} else {
		// 回退到旧行为
		envVars = append(envVars, "-e DATA_PATH=/data", "-e OUTPUT_PATH=/output")
		volumes = append(volumes, "-v /app/data:/data", "-v /app/output:/output")
	}

	// 用户自定义环境变量
	for _, ev := range task.EnvVars {
		envVars = append(envVars, fmt.Sprintf("-e %s", ev))
	}

	envStr := strings.Join(envVars, " ")
	volStr := strings.Join(volumes, " ")

	// 构建 docker run 命令
	cmd := fmt.Sprintf(
		"docker run -d --name %s --gpus all %s %s %s sh -c '%s'",
		containerName, envStr, volStr, task.Image, task.Command,
	)
	log.Printf("[task %s] SSH docker run command: %s", task.ID, cmd)

	s.emitLog(task.ID, "info", fmt.Sprintf("Starting container: %s", task.Image))
	stdout, stderr, err := sshClient.Exec(cmd)
	if err != nil {
		return fmt.Errorf("docker run failed: %v, stderr: %s", err, stderr)
	}

	task.ContainerID = strings.TrimSpace(stdout)
	s.updateTask(ctx, task)
	return nil
}

// monitorTraining 监控训练容器状态，返回退出码
func (s *TaskService) monitorTraining(ctx context.Context, task *model.Task, sshClient *executor.SSHClient, containerName string) int {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	lastLogSize := 0
	for {
		select {
		case <-ticker.C:
			// 检查容器状态
			stdout, _, err := sshClient.Exec(fmt.Sprintf("docker inspect -f '{{.State.Status}}|{{.State.ExitCode}}' %s 2>/dev/null", containerName))
			if err != nil {
				log.Printf("[task %s] Failed to inspect container: %v", task.ID, err)
				continue
			}

			parts := strings.Split(strings.TrimSpace(stdout), "|")
			if len(parts) >= 2 {
				status := parts[0]
				exitCode := 0
				fmt.Sscanf(parts[1], "%d", &exitCode)

				if status == "exited" {
					return exitCode
				}
			}

			// 获取新日志
			logs, _, _ := sshClient.Exec(fmt.Sprintf("docker logs --tail=100 %s 2>&1 | tail -c 32768", containerName))
			if len(logs) > lastLogSize {
				newLogs := logs[lastLogSize:]
				for _, line := range strings.Split(newLogs, "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						s.emitLog(task.ID, "info", line)
					}
				}
				lastLogSize = len(logs)
			}

		case <-ctx.Done():
			return -1
		}
	}
}

// startPPIOTraining 在 PPIO 实例内直接启动训练进程（不经过 Docker）
func (s *TaskService) startPPIOTraining(ctx context.Context, task *model.Task, sshClient *executor.SSHClient) (int, error) {
	// 检查 Python 可用性
	pythonCmd := "python3"
	if _, _, err := sshClient.Exec("python3 --version 2>/dev/null"); err != nil {
		if _, _, err := sshClient.Exec("python --version 2>/dev/null"); err != nil {
			return 0, fmt.Errorf("python/python3 not found on PPIO instance")
		}
		pythonCmd = "python"
	}

	// 确定训练命令
	trainCmd := task.Command
	if strings.TrimSpace(trainCmd) == "" {
		trainCmd = fmt.Sprintf("%s /app/train_mnist_cpu.py", pythonCmd)
	} else if !strings.Contains(trainCmd, "python") && strings.HasPrefix(strings.TrimSpace(trainCmd), "/app/") {
		// 命令是脚本路径但未指定解释器，自动补全
		trainCmd = fmt.Sprintf("%s %s", pythonCmd, trainCmd)
	}
	s.emitLog(task.ID, "info", fmt.Sprintf("Training command: %s", trainCmd))

	// 构建环境变量
	envExports := []string{
		fmt.Sprintf("export TASK_ID=%s", task.ID),
	}

	if len(task.StorageBindings) > 0 {
		for _, binding := range task.StorageBindings {
			localDir := fmt.Sprintf("/app/bindings/%s", binding.EnvName)
			envExports = append(envExports, fmt.Sprintf("export %s=%s", binding.EnvName, localDir))
		}
	} else {
		envExports = append(envExports, "export DATA_PATH=/app/data", "export OUTPUT_PATH=/app/output")
	}

	for _, ev := range task.EnvVars {
		envExports = append(envExports, fmt.Sprintf("export %s", ev))
	}

	envStr := strings.Join(envExports, "\n")

	// 构建启动脚本（不用 set -e，确保退出码被记录；用 set -x 输出调试信息）
	script := fmt.Sprintf(`#!/bin/bash
set -x
cd /app
%s
%s
EXIT_CODE=$?
echo $EXIT_CODE > /app/train.exitcode
exit $EXIT_CODE
`, envStr, trainCmd)

	// 写入脚本
	writeCmd := fmt.Sprintf("cat > /app/run_train.sh << 'SCRIPT_EOF'\n%s\nSCRIPT_EOF", script)
	log.Printf("[task %s] SSH write script command:\n%s", task.ID, writeCmd)
	if _, stderr, err := sshClient.Exec(writeCmd); err != nil {
		return 0, fmt.Errorf("failed to write training script: %v, stderr: %s", err, stderr)
	}
	if _, _, err := sshClient.Exec("chmod +x /app/run_train.sh"); err != nil {
		return 0, fmt.Errorf("failed to chmod script: %v", err)
	}

	// 清理旧的退出码文件
	sshClient.Exec("rm -f /app/train.exitcode /app/train.pid /app/train.log")

	// 启动训练进程
	startCmd := "nohup bash /app/run_train.sh > /app/train.log 2>&1 & echo $! > /app/train.pid"
	log.Printf("[task %s] SSH start training command: %s", task.ID, startCmd)
	if _, stderr, err := sshClient.Exec(startCmd); err != nil {
		return 0, fmt.Errorf("failed to start training process: %v, stderr: %s", err, stderr)
	}

	// 读取 PID
	stdout, _, err := sshClient.Exec("cat /app/train.pid")
	if err != nil {
		return 0, fmt.Errorf("failed to read PID: %v", err)
	}

	pid := 0
	fmt.Sscanf(strings.TrimSpace(stdout), "%d", &pid)
	if pid == 0 {
		return 0, fmt.Errorf("failed to get training PID")
	}

	s.emitLog(task.ID, "info", fmt.Sprintf("Training process started with PID %d", pid))
	return pid, nil
}

// monitorPPIOTraining 监控 PPIO 训练进程状态，返回退出码
func (s *TaskService) monitorPPIOTraining(ctx context.Context, task *model.Task, sshClient *executor.SSHClient, pid int) int {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	lastLogSize := 0

	for {
		select {
		case <-ticker.C:
			// 检查进程是否还在运行
			stdout, _, err := sshClient.Exec(fmt.Sprintf("ps -p %d -o pid= 2>/dev/null", pid))
			isRunning := err == nil && strings.TrimSpace(stdout) != ""

			if !isRunning {
				// 进程已结束，读取退出码
				exitCode := s.getPPIOExitCode(task, sshClient)
				return exitCode
			}

			// 获取新日志
			logs, _, _ := sshClient.Exec(fmt.Sprintf("tail -c +%d /app/train.log 2>/dev/null", lastLogSize+1))
			if len(logs) > 0 {
				lastLogSize += len(logs)
				for _, line := range strings.Split(logs, "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						s.emitLog(task.ID, "info", line)
					}
				}
			}

		case <-ctx.Done():
			return -1
		}
	}
}

// getPPIOExitCode 读取 PPIO 训练进程的退出码
func (s *TaskService) getPPIOExitCode(task *model.Task, sshClient *executor.SSHClient) int {
	stdout, _, err := sshClient.Exec("cat /app/train.exitcode 2>/dev/null")
	if err != nil {
		log.Printf("[task %s] Failed to read exit code: %v", task.ID, err)
		return 1
	}
	exitCode := 0
	fmt.Sscanf(strings.TrimSpace(stdout), "%d", &exitCode)
	return exitCode
}

// uploadOutputData 通过 SSH 在实例上上传输出数据到 COS
func (s *TaskService) uploadOutputData(ctx context.Context, task *model.Task, sshClient *executor.SSHClient) error {
	parts := strings.SplitN(task.OutputPath, "/", 2)
	if len(parts) < 1 {
		return fmt.Errorf("invalid output path format: %s", task.OutputPath)
	}
	bucketName := parts[0]
	prefix := task.ID + "/"
	if len(parts) > 1 && parts[1] != "" {
		prefix = parts[1] + "/" + task.ID + "/"
	}

	s.emitLog(task.ID, "info", fmt.Sprintf("Uploading output to %s/%s...", bucketName, prefix))

	// 获取实例上 /app/output 目录下的所有文件
	stdout, stderr, err := sshClient.Exec("find /app/output -type f 2>/dev/null")
	if err != nil {
		return fmt.Errorf("failed to list output files: %v, stderr: %s", err, stderr)
	}

	files := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		// 计算相对路径
		relPath := strings.TrimPrefix(file, "/app/output/")
		cosKey := prefix + relPath

		cmd := fmt.Sprintf("coscli cp -f %s %s", shellQuote(file), shellQuote(fmt.Sprintf("cos://%s/%s", bucketName, cosKey)))
		if cosRegion != "" {
			cmd += fmt.Sprintf(" -r %s", shellQuote(cosRegion))
		}
		log.Printf("[task %s] SSH upload command: %s", task.ID, cmd)
		if _, stderr, err := sshClient.Exec(cmd); err != nil {
			s.emitLog(task.ID, "warn", fmt.Sprintf("Failed to upload %s: %v", relPath, err))
			if stderr != "" {
				log.Printf("[task %s] upload stderr: %s", task.ID, stderr)
			}
		} else {
			s.emitLog(task.ID, "info", fmt.Sprintf("Uploaded %s", relPath))
		}
	}

	return nil
}

// uploadBindingData 通过 SSH 在实例上将本地目录上传到 COS 指定路径
func (s *TaskService) uploadBindingData(ctx context.Context, task *model.Task, sshClient *executor.SSHClient, cosPath string, localBaseDir string) error {
	parts := strings.SplitN(cosPath, "/", 2)
	if len(parts) < 1 {
		return fmt.Errorf("invalid path format: %s", cosPath)
	}
	bucketName := parts[0]
	prefix := task.ID + "/"
	if len(parts) > 1 && parts[1] != "" {
		prefix = parts[1] + "/" + task.ID + "/"
	}

	s.emitLog(task.ID, "info", fmt.Sprintf("Uploading %s to %s/%s...", localBaseDir, bucketName, prefix))

	stdout, stderr, err := sshClient.Exec(fmt.Sprintf("find %s -type f 2>/dev/null", localBaseDir))
	if err != nil {
		return fmt.Errorf("failed to list files: %v, stderr: %s", err, stderr)
	}

	files := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}

		relPath := strings.TrimPrefix(file, localBaseDir+"/")
		cosKey := prefix + relPath

		cmd := fmt.Sprintf("coscli cp -f %s %s", shellQuote(file), shellQuote(fmt.Sprintf("cos://%s/%s", bucketName, cosKey)))
		if cosRegion != "" {
			cmd += fmt.Sprintf(" -r %s", shellQuote(cosRegion))
		}
		log.Printf("[task %s] SSH binding upload command: %s", task.ID, cmd)
		if _, stderr, err := sshClient.Exec(cmd); err != nil {
			s.emitLog(task.ID, "warn", fmt.Sprintf("Failed to upload %s: %v", relPath, err))
			if stderr != "" {
				log.Printf("[task %s] upload stderr: %s", task.ID, stderr)
			}
		} else {
			s.emitLog(task.ID, "info", fmt.Sprintf("Uploaded %s", relPath))
		}
	}

	return nil
}

// failTask 将任务标记为失败
func (s *TaskService) failTask(ctx context.Context, task *model.Task, reason string) {
	task.Status = model.TaskStatusFailed
	task.Phase = model.TaskPhaseFailed
	task.ErrorMsg = reason
	now := time.Now()
	task.CompletedAt = &now
	s.updateTask(ctx, task)
	s.emitLog(task.ID, "error", reason)
	log.Printf("[task %s] Failed: %s", task.ID, reason)
}

// updateTask 更新任务到数据库和缓存
func (s *TaskService) updateTask(ctx context.Context, task *model.Task) {
	if err := s.repo.Update(ctx, task); err != nil {
		log.Printf("Failed to update task %s: %v", task.ID, err)
	}
	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()
}

// emitLog 发送日志到 executor
func (s *TaskService) emitLog(taskID, level, message string) {
	s.executor.EmitLog(taskID, level, message)
}

// GetTask 获取任务
func (s *TaskService) GetTask(ctx context.Context, taskID string) (*model.Task, error) {
	s.mu.RLock()
	task, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if ok {
		return task, nil
	}

	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	return task, nil
}

// ListTasks 列出所有任务
func (s *TaskService) ListTasks(ctx context.Context) ([]*model.Task, error) {
	tasks, err := s.repo.List(ctx, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	s.mu.Lock()
	for _, task := range tasks {
		s.tasks[task.ID] = task
	}
	s.mu.Unlock()

	return tasks, nil
}

// CancelTask 取消任务
func (s *TaskService) CancelTask(ctx context.Context, taskID string) error {
	s.mu.RLock()
	task, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status != model.TaskStatusPending && task.Status != model.TaskStatusRunning {
		return fmt.Errorf("task %s is not in a cancellable state", taskID)
	}

	// 如果是本地任务，调用 executor 取消
	if task.InstanceID == "" {
		if err := s.executor.CancelTask(ctx, taskID); err != nil {
			return fmt.Errorf("failed to cancel task: %w", err)
		}
	} else {
		// 远程任务：SSH 停止容器并销毁实例
		if task.SSHHost != "" && task.SSHPort > 0 {
			sshClient := executor.NewSSHClient(task.SSHHost, task.SSHPort, task.SSHUser, task.SSHPassword)
			if err := sshClient.Connect(); err == nil {
				sshClient.Exec(fmt.Sprintf("docker stop task-%s 2>/dev/null", taskID))
				sshClient.Close()
			}
		}
		// 销毁实例（同步执行，确保错误能被感知）
		instanceID := task.InstanceID
		if instanceID == "" {
			// 兜底：如果任务没有关联实例ID，尝试按名称查找
			log.Printf("[task] Task %s has empty InstanceID, trying to find instance by name", taskID)
			inst, err := s.instanceService.FindInstanceByName(ctx, fmt.Sprintf("task-%s", taskID))
			if err == nil && inst != nil {
				instanceID = inst.ID
				log.Printf("[task] Found instance %s for task %s by name", instanceID, taskID)
			}
		}
		if instanceID != "" {
			if err := s.instanceService.DestroyInstance(ctx, instanceID); err != nil {
				log.Printf("[task] Failed to destroy instance %s for task %s: %v", instanceID, taskID, err)
				return fmt.Errorf("failed to destroy remote instance: %w", err)
			}
			log.Printf("[task] Instance %s destroyed for task %s", instanceID, taskID)
		} else {
			log.Printf("[task] No instance found to destroy for task %s", taskID)
		}
	}

	task.Status = model.TaskStatusCancelled
	now := time.Now()
	task.CompletedAt = &now

	if err := s.repo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	return nil
}

// GetTaskStats 获取任务统计
func (s *TaskService) GetTaskStats(ctx context.Context) (*model.TaskStats, error) {
	return s.repo.GetStats(ctx)
}

// SubscribeLogs 订阅日志
func (s *TaskService) SubscribeLogs(taskID, clientID string) chan *model.TaskLog {
	return s.executor.SubscribeLogs(taskID, clientID)
}

// UnsubscribeLogs 取消订阅日志
func (s *TaskService) UnsubscribeLogs(taskID, clientID string) {
	s.executor.UnsubscribeLogs(taskID, clientID)
}

// dirOf 获取文件路径的目录部分
func dirOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "."
	}
	return path[:idx]
}

// shellQuote 将字符串用单引号包裹，内部单引号转义，避免 shell 解析特殊字符
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// parseCOSPath 解析 COS 路径 bucket/prefix
func parseCOSPath(path string) (bucket, prefix string, err error) {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		return "", "", fmt.Errorf("invalid COS path: %s", path)
	}
	bucket = parts[0]
	if len(parts) > 1 {
		prefix = parts[1]
	}
	return bucket, prefix, nil
}

// RemoteExecPayload 用于在实例上执行远程命令的辅助结构
type RemoteExecPayload struct {
	TaskID       string          `json:"task_id"`
	Image        string          `json:"image"`
	Command      string          `json:"command"`
	EnvVars      []string        `json:"env_vars"`
	DataPath     string          `json:"data_path"`
	OutputPath   string          `json:"output_path"`
	Downloads    []DownloadEntry `json:"downloads"`
	UploadPrefix string          `json:"upload_prefix"`
}

// DownloadEntry 预签名下载条目
type DownloadEntry struct {
	Key string `json:"key"`
	URL string `json:"url"`
}

// ToJSON 序列化为 JSON
func (p *RemoteExecPayload) ToJSON() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
