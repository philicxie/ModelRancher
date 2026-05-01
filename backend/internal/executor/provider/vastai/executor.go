package vastai

import (
	"context"
	"fmt"
	"log"
	"ml-platform/internal/executor/provider"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TaskState 任务状态
type TaskState string

const (
	TaskStatePending   TaskState = "pending"
	TaskStateStarting  TaskState = "starting"
	TaskStateRunning   TaskState = "running"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateCancelled TaskState = "cancelled"
)

// ProviderConfig Provider配置
type ProviderConfig struct {
	APIKey      string  // Vast.ai API Key
	SSHKey      string  // SSH私钥
	SSHKeyID    string  // SSH Key ID
	DefaultGPU  string  // 默认GPU类型，如 "RTX 4090"
	DefaultDisk int     // 默认磁盘大小(GB)
	MaxPrice    float64 // 最高价格($/小时)
}

// Provider executor provider实现
type Provider struct {
	client *Client
	config ProviderConfig

	// 运行中的任务
	tasks  map[string]*Task
	taskMu sync.RWMutex
}

// Task 训练任务
type Task struct {
	ID          string
	Name        string
	InstanceID  int64
	OfferID     int64
	Status      TaskState
	SSHHost     string
	SSHPort     int
	SSHClient   *SSHClient
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	ExitCode    int
	ErrorMsg    string
}

// ExecutorResult 执行结果
type ExecutorResult struct {
	Success    bool
	ExitCode   int
	OutputPath string
	Error      error
}

// NewProvider 创建Provider
func NewProvider(config ProviderConfig) (*Provider, error) {
	client, err := NewClient(Config{
		APIKey:   config.APIKey,
		SSHKey:   config.SSHKey,
		SSHKeyID: config.SSHKeyID,
	})
	if err != nil {
		return nil, err
	}

	return &Provider{
		client: client,
		config: config,
		tasks:  make(map[string]*Task),
	}, nil
}

// taskToTaskInfo 将内部Task转换为provider.TaskInfo
func taskToTaskInfo(task *Task) *provider.TaskInfo {
	info := &provider.TaskInfo{
		ID:         task.ID,
		Name:       task.Name,
		State:      provider.TaskState(task.Status),
		InstanceID: strconv.FormatInt(task.InstanceID, 10),
		Provider:   "vast.ai",
		CreatedAt:  task.CreatedAt.Format(time.RFC3339),
	}
	if task.StartedAt != nil {
		info.StartedAt = task.StartedAt.Format(time.RFC3339)
	}
	if task.CompletedAt != nil {
		info.CompletedAt = task.CompletedAt.Format(time.RFC3339)
	}
	if task.ErrorMsg != "" {
		info.Error = task.ErrorMsg
	}
	if task.SSHHost != "" {
		info.SSHHost = task.SSHHost
		info.SSHPort = task.SSHPort
	}
	return info
}

// CreateTask 创建任务
func (p *Provider) CreateTask(ctx context.Context, req *provider.CreateTaskRequest) (*provider.TaskInfo, error) {
	// 1. 搜索合适的GPU实例
	filter := OfferFilter{
		GPUName:     req.GPUType,
		NumGPUs:     req.GPUs,
		Reliability: 0.95,
		Verified:    true,
		Rentable:    true,
		Type:        "ondemand",
		Limit:       5,
	}

	if filter.GPUName == "" {
		filter.GPUName = p.config.DefaultGPU
	}

	offers, err := p.client.SearchOffers(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search offers: %w", err)
	}

	if len(offers) == 0 {
		return nil, fmt.Errorf("no available offers found")
	}

	// 选择最便宜的offer
	var selectedOffer *Offer
	for i := range offers {
		offer := &offers[i]
		if p.config.MaxPrice > 0 && offer.DPHTotal > p.config.MaxPrice {
			continue
		}
		if selectedOffer == nil || offer.DPHTotal < selectedOffer.DPHTotal {
			selectedOffer = offer
		}
	}

	if selectedOffer == nil {
		return nil, fmt.Errorf("no offers within price range")
	}

	log.Printf("[vast.ai] Selected offer: %s x%d, $%.4f/hr", selectedOffer.GPUName, selectedOffer.NumGPUs, selectedOffer.DPHTotal)

	// 2. 创建实例
	diskSize := req.DiskSize
	if diskSize == 0 {
		diskSize = p.config.DefaultDisk
	}
	if diskSize == 0 {
		diskSize = 50
	}

	createReq := CreateInstanceRequest{
		Image:   req.Image,
		Label:   "ml-task-" + req.TaskID,
		Disk:    diskSize,
		Runtype: "ssh_direct",
		Env:     req.EnvVars,
	}

	if req.SSHKey != "" {
		if createReq.Env == nil {
			createReq.Env = make(map[string]string)
		}
		createReq.Env["SSH_KEY"] = req.SSHKey
	}

	result, err := p.client.CreateInstance(ctx, selectedOffer.ID, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// 3. 创建任务记录
	task := &Task{
		ID:         req.TaskID,
		Name:       req.TaskID,
		InstanceID: result.InstanceID,
		OfferID:    selectedOffer.ID,
		Status:     TaskStateStarting,
		CreatedAt:  time.Now(),
	}

	p.taskMu.Lock()
	p.tasks[req.TaskID] = task
	p.taskMu.Unlock()

	// 4. 等待实例启动
	go p.waitAndExecute(task, req)

	return taskToTaskInfo(task), nil
}

// waitAndExecute 等待实例启动并执行任务
func (p *Provider) waitAndExecute(task *Task, req *provider.CreateTaskRequest) {
	ctx := context.Background()

	// 等待实例启动
	for i := 0; i < 60; i++ {
		instance, err := p.client.GetInstance(ctx, task.InstanceID)
		if err == nil && instance.ActualStatus == "running" {
			task.SSHHost = instance.SSHHost
			task.SSHPort = instance.SSHPort
			task.Status = TaskStateRunning
			now := time.Now()
			task.StartedAt = &now
			break
		}
		log.Printf("[vast.ai] Waiting for instance %d to start... (%d/60)", task.InstanceID, i+1)
		time.Sleep(10 * time.Second)
	}

	if task.Status != TaskStateRunning {
		task.Status = TaskStateFailed
		task.ErrorMsg = "Instance failed to start within timeout"
		return
	}

	// 连接SSH
	var err error
	task.SSHClient, err = NewSSHClient(task.SSHHost, task.SSHPort, "root", []byte(p.config.SSHKey))
	if err != nil {
		task.Status = TaskStateFailed
		task.ErrorMsg = fmt.Sprintf("SSH connection failed: %v", err)
		return
	}
	defer task.SSHClient.Close()

	// 等待SSH就绪
	if err := task.SSHClient.WaitForReady(ctx, 60*time.Second); err != nil {
		task.Status = TaskStateFailed
		task.ErrorMsg = fmt.Sprintf("SSH not ready: %v", err)
		return
	}

	log.Printf("[vast.ai] Instance %d ready, executing task...", task.InstanceID)

	// 上传数据目录
	if req.DataPath != "" {
		log.Printf("[vast.ai] Uploading data files...")
		// 实际实现中需要先下载本地数据到临时目录，再上传到远程
	}

	// 执行训练命令
	var output string
	if req.Command != "" {
		log.Printf("[vast.ai] Executing command: %s", req.Command)
		output, _, task.ExitCode, err = task.SSHClient.Execute(ctx, req.Command)
		log.Printf("[vast.ai] Command output: %s", output)
		if err != nil && task.ExitCode != 0 {
			task.Status = TaskStateFailed
			task.ErrorMsg = fmt.Sprintf("Command failed with exit code %d: %s", task.ExitCode, err)
			return
		}
	}

	// 下载输出文件
	if req.OutputPath != "" {
		log.Printf("[vast.ai] Downloading output files...")
		// 实际实现中需要从远程下载到本地
	}

	// 标记完成
	task.Status = TaskStateCompleted
	now := time.Now()
	task.CompletedAt = &now

	// 销毁实例
	log.Printf("[vast.ai] Destroying instance %d...", task.InstanceID)
	if err := p.client.DestroyInstance(ctx, task.InstanceID); err != nil {
		log.Printf("[vast.ai] Warning: failed to destroy instance: %v", err)
	}
}

// GetTask 获取任务状态
func (p *Provider) GetTask(ctx context.Context, taskID string) (*provider.TaskInfo, error) {
	p.taskMu.RLock()
	task, ok := p.tasks[taskID]
	p.taskMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return taskToTaskInfo(task), nil
}

// ListTasks 列出所有任务
func (p *Provider) ListTasks(ctx context.Context) ([]*provider.TaskInfo, error) {
	p.taskMu.RLock()
	defer p.taskMu.RUnlock()

	tasks := make([]*provider.TaskInfo, 0, len(p.tasks))
	for _, task := range p.tasks {
		tasks = append(tasks, taskToTaskInfo(task))
	}

	return tasks, nil
}

// CancelTask 取消任务
func (p *Provider) CancelTask(ctx context.Context, taskID string) error {
	p.taskMu.RLock()
	task, ok := p.tasks[taskID]
	p.taskMu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status == TaskStateCompleted || task.Status == TaskStateFailed || task.Status == TaskStateCancelled {
		return fmt.Errorf("task is not in a cancellable state")
	}

	// 关闭SSH连接
	if task.SSHClient != nil {
		task.SSHClient.Close()
	}

	// 销毁实例
	if err := p.client.DestroyInstance(ctx, task.InstanceID); err != nil {
		log.Printf("[vast.ai] Warning: failed to destroy instance: %v", err)
	}

	task.Status = TaskStateCancelled
	now := time.Now()
	task.CompletedAt = &now

	return nil
}

// ExecuteSSHCommand 在运行中的实例上执行SSH命令
func (p *Provider) ExecuteSSHCommand(ctx context.Context, taskID string, command string) (string, string, int, error) {
	p.taskMu.RLock()
	task, ok := p.tasks[taskID]
	p.taskMu.RUnlock()

	if !ok {
		return "", "", -1, fmt.Errorf("task not found: %s", taskID)
	}

	if task.SSHClient == nil {
		return "", "", -1, fmt.Errorf("SSH client not initialized")
	}

	return task.SSHClient.Execute(ctx, command)
}

// UploadFile 上传文件到实例
func (p *Provider) UploadFile(ctx context.Context, taskID string, localPath, remotePath string) error {
	p.taskMu.RLock()
	task, ok := p.tasks[taskID]
	p.taskMu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.SSHClient == nil {
		return fmt.Errorf("SSH client not initialized")
	}

	return task.SSHClient.UploadFile(ctx, localPath, remotePath)
}

// DownloadFile 从实例下载文件
func (p *Provider) DownloadFile(ctx context.Context, taskID string, remotePath, localPath string) error {
	p.taskMu.RLock()
	task, ok := p.tasks[taskID]
	p.taskMu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.SSHClient == nil {
		return fmt.Errorf("SSH client not initialized")
	}

	return task.SSHClient.DownloadFile(ctx, remotePath, localPath)
}

// ListOffers 列出可用实例
func (p *Provider) ListOffers(ctx context.Context, gpuName string, minGPUs int) ([]Offer, error) {
	filter := OfferFilter{
		GPUName:     gpuName,
		MinGPUs:     minGPUs,
		Reliability: 0.95,
		Verified:    true,
		Rentable:    true,
		Type:        "ondemand",
		Limit:       20,
	}

	return p.client.SearchOffers(ctx, filter)
}

// ListInstances 列出当前运行中的实例
func (p *Provider) ListInstances(ctx context.Context) ([]Instance, error) {
	return p.client.ShowInstances(ctx)
}

// GetExecutorStats 获取执行器统计
func (p *Provider) GetExecutorStats(ctx context.Context) *ExecutorStats {
	p.taskMu.RLock()
	defer p.taskMu.RUnlock()

	stats := &ExecutorStats{
		Total:  len(p.tasks),
		States: make(map[TaskState]int),
	}

	for _, task := range p.tasks {
		stats.States[task.Status]++
	}

	return stats
}

// ExecutorStats 执行器统计
type ExecutorStats struct {
	Total     int
	States    map[TaskState]int
	TotalCost float64 // 累计费用
}

// ProviderInfo 返回Provider信息
func (p *Provider) ProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:    "vast.ai",
		Enabled: true,
	}
}

// ProviderInfo Provider信息
type ProviderInfo struct {
	Name    string
	Enabled bool
}

// --- Lite Instance Operations ---

// liteInstances Lite实例列表
var liteInstances = make(map[string]*LiteInstance)
var liteInstancesMu sync.RWMutex

// LiteInstance Lite实例
type LiteInstance struct {
	ID         string
	InstanceID int64
	OfferID    int64
	Status     string
	Image      string
	SSHHost    string
	SSHPort    int
	SSHUser    string
	CreatedAt  time.Time
	StartedAt  *time.Time
	StoppedAt  *time.Time
}

// CreateInstance 创建Lite实例
func (p *Provider) CreateInstance(ctx context.Context, req *provider.CreateInstanceRequest) (*provider.InstanceInfo, error) {
	// 解析Offer ID
	offerID, err := strconv.ParseInt(req.OfferID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid offer ID: %w", err)
	}

	// 创建实例
	createReq := CreateInstanceRequest{
		Image:   req.Image,
		Label:   fmt.Sprintf("lite-%d", time.Now().UnixNano()),
		Disk:    req.DiskSize,
		Runtype: "ssh_direct", // SSH直接访问模式
	}

	result, err := p.client.CreateInstance(ctx, offerID, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	instanceID := fmt.Sprintf("vastai-%d", result.InstanceID)

	// 等待实例启动
	go p.waitForInstanceStartup(instanceID, result.InstanceID)

	return &provider.InstanceInfo{
		InstanceID: instanceID,
		SSHHost:    "", // 等待实例启动后填充
		SSHPort:    0,
		SSHUser:    "root",
		Status:     "starting",
	}, nil
}

// waitForInstanceStartup 等待实例启动
func (p *Provider) waitForInstanceStartup(instanceID string, providerInstanceID int64) {
	ctx := context.Background()

	for i := 0; i < 60; i++ {
		instance, err := p.client.GetInstance(ctx, providerInstanceID)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		if instance.ActualStatus == "running" {
			liteInstancesMu.Lock()
			if inst, ok := liteInstances[instanceID]; ok {
				inst.SSHHost = instance.SSHHost
				inst.SSHPort = instance.SSHPort
				inst.Status = "running"
				now := time.Now()
				inst.StartedAt = &now
			} else {
				now := time.Now()
				liteInstances[instanceID] = &LiteInstance{
					ID:         instanceID,
					InstanceID: providerInstanceID,
					Status:     "running",
					SSHHost:    instance.SSHHost,
					SSHPort:    instance.SSHPort,
					SSHUser:    "root",
					CreatedAt:  time.Now(),
					StartedAt:  &now,
				}
			}
			liteInstancesMu.Unlock()
			return
		}

		time.Sleep(10 * time.Second)
	}
}

// StopInstance 停止Lite实例
func (p *Provider) StopInstance(ctx context.Context, instanceID string) error {
	liteInstancesMu.RLock()
	instance, ok := liteInstances[instanceID]
	liteInstancesMu.RUnlock()

	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	if err := p.client.StopInstance(ctx, instance.InstanceID); err != nil {
		return err
	}

	liteInstancesMu.Lock()
	instance.Status = "stopped"
	now := time.Now()
	instance.StoppedAt = &now
	liteInstancesMu.Unlock()

	return nil
}

// StartInstance 启动Lite实例
func (p *Provider) StartInstance(ctx context.Context, instanceID string) error {
	liteInstancesMu.RLock()
	instance, ok := liteInstances[instanceID]
	liteInstancesMu.RUnlock()

	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	if err := p.client.StartInstance(ctx, instance.InstanceID); err != nil {
		return err
	}

	liteInstancesMu.Lock()
	instance.Status = "starting"
	liteInstancesMu.Unlock()

	// 异步等待启动
	go p.waitForInstanceStartup(instanceID, instance.InstanceID)

	return nil
}

// DestroyInstance 销毁Lite实例
func (p *Provider) DestroyInstance(ctx context.Context, instanceID string) error {
	liteInstancesMu.RLock()
	instance, ok := liteInstances[instanceID]
	liteInstancesMu.RUnlock()

	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	if err := p.client.DestroyInstance(ctx, instance.InstanceID); err != nil {
		return err
	}

	liteInstancesMu.Lock()
	delete(liteInstances, instanceID)
	liteInstancesMu.Unlock()

	return nil
}

// GetInstanceStatus 获取实例状态
func (p *Provider) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	liteInstancesMu.RLock()
	instance, ok := liteInstances[instanceID]
	liteInstancesMu.RUnlock()

	if !ok {
		return "", fmt.Errorf("instance not found: %s", instanceID)
	}

	// 查询远程状态
	remoteInstance, err := p.client.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		return instance.Status, nil
	}

	return remoteInstance.ActualStatus, nil
}

// GetInstance 获取实例详情（含SSH信息）
func (p *Provider) GetInstance(ctx context.Context, instanceID string) (*provider.InstanceInfo, error) {
	liteInstancesMu.RLock()
	inst, ok := liteInstances[instanceID]
	liteInstancesMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	info := &provider.InstanceInfo{
		InstanceID: instanceID,
		SSHHost:    inst.SSHHost,
		SSHPort:    inst.SSHPort,
		SSHUser:    inst.SSHUser,
		Status:     inst.Status,
	}

	// 查询远程获取最新状态
	remoteInstance, err := p.client.GetInstance(ctx, inst.InstanceID)
	if err == nil {
		info.SSHHost = remoteInstance.SSHHost
		info.SSHPort = remoteInstance.SSHPort
		info.SSHUser = "root"
		info.Status = remoteInstance.ActualStatus
	}

	return info, nil
}

// ListAvailableResources 列出可用计算资源
func (p *Provider) ListAvailableResources(ctx context.Context, req *provider.ResourceRequest) ([]*provider.Resource, error) {
	filter := OfferFilter{
		GPUName:     req.GPUType,
		NumGPUs:     req.MinGPUs,
		Reliability: 0.95,
		Verified:    true,
		Rentable:    true,
		Type:        "ondemand",
		Limit:       50,
	}

	offers, err := p.client.SearchOffers(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search offers: %w", err)
	}

	resources := make([]*provider.Resource, 0, len(offers))
	for i := range offers {
		offer := &offers[i]
		if req.MaxPrice > 0 && offer.DPHTotal > req.MaxPrice {
			continue
		}
		resources = append(resources, &provider.Resource{
			ID:          fmt.Sprintf("%d", offer.ID),
			Provider:    "vast.ai",
			GPUType:     offer.GPUName,
			NumGPUs:     offer.NumGPUs,
			GPURAM:      offer.GPURAM,
			DiskSpace:   offer.DiskSpace,
			Reliability: offer.Reliability,
			Price:       offer.DPHTotal,
			Location:    offer.Country,
		})
	}

	return resources, nil
}

// GetStats 获取统计信息
func (p *Provider) GetStats(ctx context.Context) (*provider.Stats, error) {
	p.taskMu.RLock()
	defer p.taskMu.RUnlock()

	stats := &provider.Stats{
		TotalTasks:      len(p.tasks),
		TasksByProvider: map[string]int{"vast.ai": len(p.tasks)},
	}
	for _, task := range p.tasks {
		switch task.Status {
		case TaskStateRunning:
			stats.RunningTasks++
		case TaskStateCompleted:
			stats.CompletedTasks++
		case TaskStateFailed:
			stats.FailedTasks++
		}
	}
	return stats, nil
}

// HealthCheck 健康检查
func (p *Provider) HealthCheck(ctx context.Context) error {
	_, err := p.client.SearchOffers(ctx, OfferFilter{Limit: 1})
	return err
}

// ParseSSHKey 从字符串解析SSH密钥
func ParseSSHKey(keyData string) (string, string, error) {
	lines := strings.Split(keyData, "\n")
	var keyType, keyBody string

	for _, line := range lines {
		if strings.HasPrefix(line, "-----BEGIN") {
			keyType = "rsa" // 默认
			if strings.Contains(line, "ED25519") {
				keyType = "ed25519"
			} else if strings.Contains(line, "RSA") {
				keyType = "rsa"
			}
		}
	}

	keyBody = strings.Join(lines, "")

	return keyType, keyBody, nil
}
