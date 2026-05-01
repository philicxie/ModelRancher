package dummy

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"ml-platform/internal/executor"
	"ml-platform/internal/executor/provider"
	"ml-platform/internal/model"

	"github.com/google/uuid"
)

// Provider 本地Dummy Provider，使用本地Docker运行任务
type Provider struct {
	executor  *executor.Executor
	tasks     map[string]*provider.TaskInfo
	taskMu    sync.RWMutex
	instances map[string]*instanceRecord
	instMu    sync.RWMutex
}

// instanceRecord Lite实例记录
type instanceRecord struct {
	info      *provider.InstanceInfo
	container string
	status    string
	createdAt time.Time
}

// NewProvider 创建Dummy Provider
func NewProvider(exec *executor.Executor) *Provider {
	return &Provider{
		executor:  exec,
		tasks:     make(map[string]*provider.TaskInfo),
		instances: make(map[string]*instanceRecord),
	}
}

// ============================================================================
// 训练任务接口
// ============================================================================

// CreateTask 创建训练任务（本地Docker异步执行）
func (p *Provider) CreateTask(ctx context.Context, req *provider.CreateTaskRequest) (*provider.TaskInfo, error) {
	taskID := req.TaskID
	if taskID == "" {
		taskID = uuid.New().String()
	}

	task := &model.Task{
		ID:        taskID,
		Name:      req.Name,
		Image:     req.Image,
		Command:   req.Command,
		Status:    model.TaskStatusPending,
		CreatedAt: time.Now(),
	}

	info := &provider.TaskInfo{
		ID:        taskID,
		Name:      req.Name,
		State:     provider.TaskStatePending,
		Provider:  "local",
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	p.taskMu.Lock()
	p.tasks[taskID] = info
	p.taskMu.Unlock()

	// 异步执行本地Docker任务
	go p.runTask(task, info)

	return info, nil
}

func (p *Provider) runTask(task *model.Task, info *provider.TaskInfo) {
	info.State = provider.TaskStateRunning
	now := time.Now()
	info.StartedAt = now.Format(time.RFC3339)

	err := p.executor.ExecuteTask(context.Background(), task)

	p.taskMu.Lock()
	defer p.taskMu.Unlock()

	if task.Status == model.TaskStatusCompleted {
		info.State = provider.TaskStateCompleted
	} else if task.Status == model.TaskStatusFailed {
		info.State = provider.TaskStateFailed
		info.Error = task.ErrorMsg
	} else if task.Status == model.TaskStatusCancelled {
		info.State = provider.TaskStateCancelled
	}
	info.CompletedAt = time.Now().Format(time.RFC3339)

	if err != nil {
		log.Printf("[dummy] Task %s finished with error: %v", task.ID, err)
	} else {
		log.Printf("[dummy] Task %s finished successfully", task.ID)
	}
}

// GetTask 获取任务状态
func (p *Provider) GetTask(ctx context.Context, taskID string) (*provider.TaskInfo, error) {
	p.taskMu.RLock()
	defer p.taskMu.RUnlock()

	info, ok := p.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return info, nil
}

// ListTasks 列出所有任务
func (p *Provider) ListTasks(ctx context.Context) ([]*provider.TaskInfo, error) {
	p.taskMu.RLock()
	defer p.taskMu.RUnlock()

	tasks := make([]*provider.TaskInfo, 0, len(p.tasks))
	for _, info := range p.tasks {
		tasks = append(tasks, info)
	}
	return tasks, nil
}

// CancelTask 取消任务
func (p *Provider) CancelTask(ctx context.Context, taskID string) error {
	return p.executor.CancelTask(ctx, taskID)
}

// UploadFile 上传文件到本地任务目录
func (p *Provider) UploadFile(ctx context.Context, taskID, localPath, remotePath string) error {
	dataDir := filepath.Join("/app/data", taskID)
	target := filepath.Join(dataDir, remotePath)
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}
	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// DownloadFile 从本地任务目录下载文件
func (p *Provider) DownloadFile(ctx context.Context, taskID, remotePath, localPath string) error {
	outputDir := filepath.Join("/app/output", taskID)
	src := filepath.Join(outputDir, remotePath)
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// ExecuteSSHCommand 在本地执行命令
func (p *Provider) ExecuteSSHCommand(ctx context.Context, taskID, command string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	stdout := string(output)
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return stdout, "", exitCode, nil
}

// ============================================================================
// 资源与统计
// ============================================================================

// ListAvailableResources 列出本地可用资源
func (p *Provider) ListAvailableResources(ctx context.Context, req *provider.ResourceRequest) ([]*provider.Resource, error) {
	resources := []*provider.Resource{
		{
			ID:          "local-docker",
			Provider:    "local",
			GPUType:     "CPU/Docker",
			NumGPUs:     0,
			GPURAM:      0,
			DiskSpace:   1000,
			Reliability: 1.0,
			Price:       0,
			Location:    "本地",
		},
	}

	// 检查是否可以使用GPU
	if hasGPU() {
		resources = append(resources, &provider.Resource{
			ID:          "local-gpu",
			Provider:    "local",
			GPUType:     "Local GPU",
			NumGPUs:     1,
			GPURAM:      0,
			DiskSpace:   1000,
			Reliability: 1.0,
			Price:       0,
			Location:    "本地",
		})
	}

	return resources, nil
}

func hasGPU() bool {
	cmd := exec.Command("nvidia-smi", "-L")
	err := cmd.Run()
	return err == nil
}

// GetStats 获取统计信息
func (p *Provider) GetStats(ctx context.Context) (*provider.Stats, error) {
	p.taskMu.RLock()
	defer p.taskMu.RUnlock()

	stats := &provider.Stats{
		TotalTasks:      len(p.tasks),
		TasksByProvider: map[string]int{"local": len(p.tasks)},
	}
	for _, t := range p.tasks {
		switch t.State {
		case provider.TaskStateRunning:
			stats.RunningTasks++
		case provider.TaskStateCompleted:
			stats.CompletedTasks++
		case provider.TaskStateFailed:
			stats.FailedTasks++
		}
	}
	return stats, nil
}

// HealthCheck 健康检查（检查Docker可用性）
func (p *Provider) HealthCheck(ctx context.Context) error {
	_, err := os.Stat("/var/run/docker.sock")
	if err != nil {
		return fmt.Errorf("docker socket not found: %w", err)
	}
	cmd := exec.CommandContext(ctx, "docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker daemon not accessible: %w", err)
	}
	return nil
}

// ============================================================================
// Lite Instance 操作
// ============================================================================

// CreateInstance 创建Lite实例（本地Docker容器）
func (p *Provider) CreateInstance(ctx context.Context, req *provider.CreateInstanceRequest) (*provider.InstanceInfo, error) {
	instanceID := fmt.Sprintf("local-%d", time.Now().UnixNano())
	containerName := fmt.Sprintf("ml-instance-%s", instanceID)

	// 创建持久化数据目录
	dataDir := filepath.Join("/app/data", "instances", instanceID)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create instance data dir: %w", err)
	}

	// 启动一个后台Docker容器
	image := req.Image
	if image == "" {
		image = "ubuntu:22.04"
	}
	cmd := exec.CommandContext(ctx, "docker", "run", "-d", "--name", containerName,
		"-v", fmt.Sprintf("%s:/workspace", dataDir),
		image, "sleep", "infinity")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w, output: %s", err, string(output))
	}

	containerID := string(output)
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	info := &provider.InstanceInfo{
		InstanceID: instanceID,
		SSHHost:    "localhost",
		SSHPort:    0,
		SSHUser:    "root",
		Status:     "running",
	}

	p.instMu.Lock()
	p.instances[instanceID] = &instanceRecord{
		info:      info,
		container: containerName,
		status:    "running",
		createdAt: time.Now(),
	}
	p.instMu.Unlock()

	log.Printf("[dummy] Created local instance %s (container: %s)", instanceID, containerName)
	return info, nil
}

// StopInstance 停止Lite实例
func (p *Provider) StopInstance(ctx context.Context, instanceID string) error {
	p.instMu.RLock()
	rec, ok := p.instances[instanceID]
	p.instMu.RUnlock()
	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	cmd := exec.CommandContext(ctx, "docker", "stop", rec.container)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container: %w, output: %s", err, string(output))
	}

	p.instMu.Lock()
	rec.status = "stopped"
	rec.info.Status = "stopped"
	p.instMu.Unlock()
	return nil
}

// StartInstance 启动Lite实例
func (p *Provider) StartInstance(ctx context.Context, instanceID string) error {
	p.instMu.RLock()
	rec, ok := p.instances[instanceID]
	p.instMu.RUnlock()
	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	cmd := exec.CommandContext(ctx, "docker", "start", rec.container)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %w, output: %s", err, string(output))
	}

	p.instMu.Lock()
	rec.status = "running"
	rec.info.Status = "running"
	p.instMu.Unlock()
	return nil
}

// DestroyInstance 销毁Lite实例
func (p *Provider) DestroyInstance(ctx context.Context, instanceID string) error {
	p.instMu.RLock()
	rec, ok := p.instances[instanceID]
	p.instMu.RUnlock()
	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", rec.container)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[dummy] Warning: failed to remove container: %v, output: %s", err, string(output))
	}

	p.instMu.Lock()
	delete(p.instances, instanceID)
	p.instMu.Unlock()
	return nil
}

// GetInstance 获取实例详情
func (p *Provider) GetInstance(ctx context.Context, instanceID string) (*provider.InstanceInfo, error) {
	p.instMu.RLock()
	rec, ok := p.instances[instanceID]
	p.instMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}
	return rec.info, nil
}

// GetInstanceStatus 获取实例状态
func (p *Provider) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	p.instMu.RLock()
	rec, ok := p.instances[instanceID]
	p.instMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("instance not found: %s", instanceID)
	}

	// 查询容器实际状态
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", rec.container)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return rec.status, nil
	}

	actualStatus := string(output)
	if len(actualStatus) > 0 && actualStatus[len(actualStatus)-1] == '\n' {
		actualStatus = actualStatus[:len(actualStatus)-1]
	}

	p.instMu.Lock()
	rec.status = actualStatus
	rec.info.Status = actualStatus
	p.instMu.Unlock()

	return actualStatus, nil
}
