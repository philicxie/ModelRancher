package dummy

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ml-platform/internal/executor"
	"ml-platform/internal/executor/provider"
	"ml-platform/internal/model"

	"github.com/google/uuid"
)

// Provider 本地Dummy Provider，纯内存模拟，不依赖Docker
type Provider struct {
	executor  *executor.Executor
	tasks     map[string]*provider.TaskInfo
	taskMu    sync.RWMutex
	instances map[string]*instanceRecord
	instMu    sync.RWMutex
}

// instanceRecord 实例记录
type instanceRecord struct {
	info      *provider.InstanceInfo
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

// CreateTask 创建训练任务
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
	dataDir := filepath.Join("/tmp", "ml-data", taskID)
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
	outputDir := filepath.Join("/tmp", "ml-output", taskID)
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

// ExecuteSSHCommand 执行命令（本地回退）
func (p *Provider) ExecuteSSHCommand(ctx context.Context, taskID, command string) (string, string, int, error) {
	// 尝试作为 instanceID 在容器内执行（兼容实例模式）
	p.instMu.RLock()
	rec, ok := p.instances[taskID]
	p.instMu.RUnlock()

	if ok {
		// 模拟容器内执行
		return fmt.Sprintf("[dummy exec %s] %s", rec.info.InstanceID, command), "", 0, nil
	}

	// 回退到宿主机执行
	return fmt.Sprintf("[dummy local] %s", command), "", 0, nil
}

// ============================================================================
// 资源与统计
// ============================================================================

// ListAvailableResources 列出本地可用资源
func (p *Provider) ListAvailableResources(ctx context.Context, req *provider.ResourceRequest) ([]*provider.Resource, error) {
	return []*provider.Resource{
		{
			ID:          "local-docker",
			Provider:    "local",
			GPUType:     "CPU/Docker",
			NumGPUs:     0,
			GPURAM:      0,
			DiskSpace:   1000,
			Reliability: 1.0,
			Price:       0.05,
			Location:    "本地",
		},
		{
			ID:          "local-gpu",
			Provider:    "local",
			GPUType:     "RTX 4090",
			NumGPUs:     1,
			GPURAM:      24,
			DiskSpace:   1000,
			Reliability: 1.0,
			Price:       0.15,
			Location:    "本地",
		},
	}, nil
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

// HealthCheck 健康检查（始终通过）
func (p *Provider) HealthCheck(ctx context.Context) error {
	return nil
}

// ============================================================================
// Lite Instance 操作（纯内存模拟）
// ============================================================================

// CreateInstance 创建实例（即刻成功，纯内存）
func (p *Provider) CreateInstance(ctx context.Context, req *provider.CreateInstanceRequest) (*provider.InstanceInfo, error) {
	instanceID := fmt.Sprintf("local-%d", time.Now().UnixNano())
	sshPort := 22000 + rand.Intn(1000)
	password := randomPassword(12)

	info := &provider.InstanceInfo{
		InstanceID: instanceID,
		SSHHost:    "localhost",
		SSHPort:    sshPort,
		SSHUser:    "root",
		Password:   password,
		SSHCommand: fmt.Sprintf("ssh root@localhost -p %d", sshPort),
		Status:     "running",
	}

	p.instMu.Lock()
	p.instances[instanceID] = &instanceRecord{
		info:      info,
		status:    "running",
		createdAt: time.Now(),
	}
	p.instMu.Unlock()

	log.Printf("[dummy] Created instance %s (ssh: %s)", instanceID, info.SSHCommand)
	return info, nil
}

// randomPassword 生成随机密码
func randomPassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// StopInstance 停止实例（仅改内存状态）
func (p *Provider) StopInstance(ctx context.Context, instanceID string) error {
	p.instMu.Lock()
	defer p.instMu.Unlock()

	rec, ok := p.instances[instanceID]
	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}
	rec.status = "stopped"
	rec.info.Status = "stopped"
	log.Printf("[dummy] Stopped instance %s", instanceID)
	return nil
}

// StartInstance 启动实例（仅改内存状态）
func (p *Provider) StartInstance(ctx context.Context, instanceID string) error {
	p.instMu.Lock()
	defer p.instMu.Unlock()

	rec, ok := p.instances[instanceID]
	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}
	rec.status = "running"
	rec.info.Status = "running"
	log.Printf("[dummy] Started instance %s", instanceID)
	return nil
}

// DestroyInstance 销毁实例（从内存中删除）
func (p *Provider) DestroyInstance(ctx context.Context, instanceID string) error {
	p.instMu.Lock()
	defer p.instMu.Unlock()

	_, ok := p.instances[instanceID]
	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}
	delete(p.instances, instanceID)
	log.Printf("[dummy] Destroyed instance %s", instanceID)
	return nil
}

// GetInstance 获取实例详情
func (p *Provider) GetInstance(ctx context.Context, instanceID string) (*provider.InstanceInfo, error) {
	p.instMu.RLock()
	defer p.instMu.RUnlock()

	rec, ok := p.instances[instanceID]
	if !ok {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}
	return rec.info, nil
}

// GetInstanceMetrics 获取实例监控指标（模拟数据）
func (p *Provider) GetInstanceMetrics(ctx context.Context, instanceID string, startTime, endTime int64) (*provider.InstanceMetrics, error) {
	p.instMu.RLock()
	_, ok := p.instances[instanceID]
	p.instMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	now := time.Now().Unix()
	randFloat := func(min, max float64) float64 {
		return min + (max-min)*float64(now%100)/100
	}

	return &provider.InstanceMetrics{
		CPUUtilization:       []provider.MetricPoint{{Timestamp: now, Value: randFloat(10, 60)}},
		MemUtilization:       []provider.MetricPoint{{Timestamp: now, Value: randFloat(20, 70)}},
		RootDiskUtilization:  []provider.MetricPoint{{Timestamp: now, Value: randFloat(5, 30)}},
		GPUUtilizationAvg:    []provider.MetricPoint{{Timestamp: now, Value: randFloat(0, 90)}},
		GPUUtilization:       []provider.GPUInstanceMetrics{{GPUID: "0", Items: []provider.MetricPoint{{Timestamp: now, Value: randFloat(0, 90)}}}},
		GPUMemUtilizationAvg: []provider.MetricPoint{{Timestamp: now, Value: randFloat(10, 80)}},
		GPUMemUtilization:    []provider.GPUInstanceMetrics{{GPUID: "0", Items: []provider.MetricPoint{{Timestamp: now, Value: randFloat(10, 80)}}}},
	}, nil
}

// GetInstanceStatus 获取实例状态
func (p *Provider) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	p.instMu.RLock()
	defer p.instMu.RUnlock()

	rec, ok := p.instances[instanceID]
	if !ok {
		return "", fmt.Errorf("instance not found: %s", instanceID)
	}
	return rec.status, nil
}
