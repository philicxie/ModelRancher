package ppio

import (
	"context"
	"fmt"
	"log"
	"ml-platform/internal/executor/provider"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProviderConfig Provider配置
type ProviderConfig struct {
	APIKey      string
	DefaultGPU  string
	DefaultDisk int
	MaxPrice    float64
}

// Provider PPIO executor provider实现
type Provider struct {
	client    *Client
	config    ProviderConfig
	tasks     map[string]*ppioTask
	taskMu    sync.RWMutex
	instances map[string]*ppioInstance
	instMu    sync.RWMutex
}

// ppioTask 训练任务
type ppioTask struct {
	ID          string
	Name        string
	InstanceID  string
	Status      provider.TaskState
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	ErrorMsg    string
}

// ppioInstance Lite实例记录
type ppioInstance struct {
	ID         string
	ProviderID string
	Status     string
	SSHHost    string
	SSHPort    int
	SSHUser    string
	Image      string
	CreatedAt  time.Time
	StartedAt  *time.Time
	StoppedAt  *time.Time
}

// NewProvider 创建PPIO Provider
func NewProvider(config ProviderConfig) (*Provider, error) {
	client, err := NewClient(Config{APIKey: config.APIKey})
	if err != nil {
		return nil, err
	}

	if config.DefaultGPU == "" {
		config.DefaultGPU = "RTX 4090"
	}
	if config.DefaultDisk == 0 {
		config.DefaultDisk = 50
	}
	if config.MaxPrice == 0 {
		config.MaxPrice = 10.0
	}

	return &Provider{
		client:    client,
		config:    config,
		tasks:     make(map[string]*ppioTask),
		instances: make(map[string]*ppioInstance),
	}, nil
}

// ============================================================================
// 训练任务接口（PPIO 无直接训练任务 API，暂以创建实例+执行命令方式实现核心逻辑）
// ============================================================================

// CreateTask 创建训练任务
func (p *Provider) CreateTask(ctx context.Context, req *provider.CreateTaskRequest) (*provider.TaskInfo, error) {
	// 1. 查找合适的产品
	products, err := p.client.ListProducts(ctx, req.GPUType, req.GPUs)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	var selected *Product
	for i := range products {
		prod := &products[i]
		if !prod.AvailableDeploy {
			continue
		}
		if p.config.MaxPrice > 0 && prod.Price.Float64() > p.config.MaxPrice {
			continue
		}
		if selected == nil || prod.Price.Float64() < selected.Price.Float64() {
			selected = prod
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("no available product found")
	}

	// 2. 创建GPU实例
	diskSize := req.DiskSize
	if diskSize == 0 {
		diskSize = p.config.DefaultDisk
	}

	createReq := CreateGPUInstanceRequest{
		Name:        "ml-task-" + req.TaskID,
		ProductID:   selected.ID,
		GpuNum:      req.GPUs,
		RootfsSize:  diskSize,
		ImageUrl:    req.Image,
		BillingMode: "postpaid",
		Kind:        "onDemand",
	}

	if req.Command != "" {
		createReq.Command = req.Command
	}
	if len(req.EnvVars) > 0 {
		envs := make([]Env, 0, len(req.EnvVars))
		for k, v := range req.EnvVars {
			envs = append(envs, Env{Key: k, Value: v})
		}
		createReq.Envs = envs
	}

	result, err := p.client.CreateGPUInstance(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// 3. 创建任务记录
	task := &ppioTask{
		ID:         req.TaskID,
		Name:       req.Name,
		InstanceID: result.InstanceID(),
		Status:     provider.TaskStateStarting,
		CreatedAt:  time.Now(),
	}

	p.taskMu.Lock()
	p.tasks[req.TaskID] = task
	p.taskMu.Unlock()

	// 4. 异步等待实例启动并跟踪状态
	go p.waitAndExecute(task, req)

	return p.taskToInfo(task), nil
}

func (p *Provider) waitAndExecute(task *ppioTask, req *provider.CreateTaskRequest) {
	ctx := context.Background()

	// 等待实例启动（最多10分钟）
	for i := 0; i < 60; i++ {
		inst, err := p.client.GetGPUInstance(ctx, task.InstanceID)
		if err == nil && inst.Status == "running" {
			now := time.Now()
			task.StartedAt = &now
			task.Status = provider.TaskStateRunning
			break
		}
		log.Printf("[ppio] Waiting for instance %s to start... (%d/60)", task.InstanceID, i+1)
		time.Sleep(10 * time.Second)
	}

	if task.Status != provider.TaskStateRunning {
		task.Status = provider.TaskStateFailed
		task.ErrorMsg = "Instance failed to start within timeout"
		return
	}

	// PPIO 实例在创建时已通过 command 字段执行了训练命令
	// 这里通过查询实例状态来判断任务是否完成
	// 简单处理：标记为运行中，由外部轮询获取状态
	log.Printf("[ppio] Instance %s is running, task %s executing...", task.InstanceID, task.ID)

	// 轮询检查实例状态（若实例被删除或停止则认为任务完成）
	for i := 0; i < 360; i++ { // 最多1小时
		time.Sleep(10 * time.Second)
		inst, err := p.client.GetGPUInstance(ctx, task.InstanceID)
		if err != nil {
			log.Printf("[ppio] Instance %s query error: %v", task.InstanceID, err)
			continue
		}
		if inst.Status == "stopped" || inst.Status == "deleted" {
			now := time.Now()
			task.CompletedAt = &now
			task.Status = provider.TaskStateCompleted
			log.Printf("[ppio] Task %s completed (instance status: %s)", task.ID, inst.Status)
			return
		}
	}

	// 超时标记完成
	now := time.Now()
	task.CompletedAt = &now
	task.Status = provider.TaskStateCompleted
	log.Printf("[ppio] Task %s marked completed after long running", task.ID)
}

// GetTask 获取任务状态
func (p *Provider) GetTask(ctx context.Context, taskID string) (*provider.TaskInfo, error) {
	p.taskMu.RLock()
	task, ok := p.tasks[taskID]
	p.taskMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return p.taskToInfo(task), nil
}

// ListTasks 列出所有任务
func (p *Provider) ListTasks(ctx context.Context) ([]*provider.TaskInfo, error) {
	p.taskMu.RLock()
	defer p.taskMu.RUnlock()

	infos := make([]*provider.TaskInfo, 0, len(p.tasks))
	for _, task := range p.tasks {
		infos = append(infos, p.taskToInfo(task))
	}
	return infos, nil
}

// CancelTask 取消任务
func (p *Provider) CancelTask(ctx context.Context, taskID string) error {
	p.taskMu.RLock()
	task, ok := p.tasks[taskID]
	p.taskMu.RUnlock()
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status == provider.TaskStateCompleted || task.Status == provider.TaskStateFailed || task.Status == provider.TaskStateCancelled {
		return fmt.Errorf("task is not in a cancellable state")
	}

	// 删除实例
	if err := p.client.DeleteGPUInstance(ctx, task.InstanceID); err != nil {
		log.Printf("[ppio] Warning: failed to delete instance %s: %v", task.InstanceID, err)
	}

	task.Status = provider.TaskStateCancelled
	now := time.Now()
	task.CompletedAt = &now
	return nil
}

// UploadFile 上传文件到实例（未实现，需通过 SSH/SFTP）
func (p *Provider) UploadFile(ctx context.Context, taskID, localPath, remotePath string) error {
	return fmt.Errorf("upload file not implemented for ppio provider")
}

// DownloadFile 从实例下载文件（未实现，需通过 SSH/SFTP）
func (p *Provider) DownloadFile(ctx context.Context, taskID, remotePath, localPath string) error {
	return fmt.Errorf("download file not implemented for ppio provider")
}

// ExecuteSSHCommand 在实例上执行SSH命令（未实现）
func (p *Provider) ExecuteSSHCommand(ctx context.Context, taskID, command string) (string, string, int, error) {
	return "", "", -1, fmt.Errorf("execute ssh command not implemented for ppio provider")
}

func (p *Provider) taskToInfo(task *ppioTask) *provider.TaskInfo {
	info := &provider.TaskInfo{
		ID:         task.ID,
		Name:       task.Name,
		State:      task.Status,
		InstanceID: task.InstanceID,
		Provider:   "ppio",
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
	return info
}

// ============================================================================
// 资源与统计
// ============================================================================

// ListAvailableResources 列出可用计算资源
func (p *Provider) ListAvailableResources(ctx context.Context, req *provider.ResourceRequest) ([]*provider.Resource, error) {
	gpuNum := req.MinGPUs
	if gpuNum == 0 {
		gpuNum = 1
	}

	products, err := p.client.ListProducts(ctx, req.GPUType, gpuNum)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	fmt.Printf("ppio list origin: %+v", products[0])
	resources := make([]*provider.Resource, 0, len(products))
	for i := range products {
		prod := &products[i]
		if !prod.AvailableDeploy {
			continue
		}

		// 解析 spotPrice（人民币）
		spotPrice := 0.0
		if prod.SpotPrice != "" {
			spotPrice, _ = strconv.ParseFloat(prod.SpotPrice, 64)
		}

		price := prod.Price.Float64()
		if spotPrice > 0 && spotPrice < price {
			price = spotPrice
		}

		// PPIO 返回的价格需先除以 100000 才是人民币元，再按固定汇率 7 CNY = 1 USD 转为美元
		const cnyToUsd = 7.0
		price = price / 100000.0 / cnyToUsd

		// 按美元价格过滤
		if req.MaxPrice > 0 && price > req.MaxPrice {
			continue
		}

		// 从 regions 中提取位置信息（取第一个 region 的 value）
		location := ""
		for _, regionName := range prod.Regions {
			location = regionName
			break
		}
		if location == "" {
			location = "PPIO"
		}

		resources = append(resources, &provider.Resource{
			ID:          prod.ID,
			Provider:    "ppio",
			GPUType:     prod.Name,
			NumGPUs:     gpuNum,
			GPURAM:      prod.MemoryPerGpu,
			DiskSpace:   float64(prod.DiskPerGpu),
			Reliability: 1.0,
			Price:       price,
			Location:    location,
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
		TasksByProvider: map[string]int{"ppio": len(p.tasks)},
	}
	for _, task := range p.tasks {
		switch task.Status {
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

// HealthCheck 健康检查
func (p *Provider) HealthCheck(ctx context.Context) error {
	_, err := p.client.ListProducts(ctx, "", 1)
	return err
}

// ============================================================================
// Lite Instance 操作
// ============================================================================

// CreateInstance 创建Lite实例
func (p *Provider) CreateInstance(ctx context.Context, req *provider.CreateInstanceRequest) (*provider.InstanceInfo, error) {
	productID := req.OfferID
	gpuNum := req.NumGPUs
	if gpuNum == 0 {
		gpuNum = 1
	}

	diskSize := req.DiskSize
	if diskSize == 0 {
		diskSize = p.config.DefaultDisk
	}

	// 查询产品信息以获取 clusterId 和系统盘限制
	products, err := p.client.ListProducts(ctx, "", gpuNum)
	if err != nil {
		log.Printf("[ppio] Warning: failed to list products for clusterId: %v", err)
	}

	var billingMode string
	for _, prod := range products {
		if prod.ID == productID {
			// 确保 rootfsSize 在合法范围内
			if prod.MinRootFS > 0 && diskSize < prod.MinRootFS {
				diskSize = prod.MinRootFS
			}
			if prod.MaxRootFS > 0 && diskSize > prod.MaxRootFS {
				diskSize = prod.MaxRootFS
			}
			// 根据产品支持的计费方式选择（优先 onDemand，资源更稳定）
			for _, bm := range prod.BillingMethods {
				if bm == "onDemand" {
					billingMode = "onDemand"
					break
				}
				if bm == "spot" && billingMode == "" {
					billingMode = "spot"
				}
			}
			break
		}
	}
	if billingMode == "" {
		billingMode = "onDemand"
	}

	createReq := CreateGPUInstanceRequest{
		Name:        fmt.Sprintf("ml-instance-%d", time.Now().UnixNano()),
		ProductID:   productID,
		GpuNum:      gpuNum,
		RootfsSize:  diskSize,
		ImageUrl:    req.Image,
		Ports:       "22/tcp",
		BillingMode: billingMode,
	}

	result, err := p.client.CreateGPUInstance(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// 记录实例
	p.instMu.Lock()
	p.instances[result.InstanceID()] = &ppioInstance{
		ID:         result.InstanceID(),
		ProviderID: result.InstanceID(),
		Status:     "creating",
		Image:      req.Image,
		CreatedAt:  time.Now(),
	}
	p.instMu.Unlock()

	return &provider.InstanceInfo{
		InstanceID: result.InstanceID(),
		Status:     "creating",
		SSHUser:    "root",
	}, nil
}

// StopInstance 停止Lite实例
func (p *Provider) StopInstance(ctx context.Context, instanceID string) error {
	if err := p.client.StopGPUInstance(ctx, instanceID); err != nil {
		return err
	}

	p.instMu.Lock()
	if inst, ok := p.instances[instanceID]; ok {
		inst.Status = "stopped"
		now := time.Now()
		inst.StoppedAt = &now
	}
	p.instMu.Unlock()

	return nil
}

// StartInstance 启动Lite实例
func (p *Provider) StartInstance(ctx context.Context, instanceID string) error {
	if err := p.client.StartGPUInstance(ctx, instanceID); err != nil {
		return err
	}

	p.instMu.Lock()
	if inst, ok := p.instances[instanceID]; ok {
		inst.Status = "starting"
		now := time.Now()
		inst.StartedAt = &now
	}
	p.instMu.Unlock()

	return nil
}

// DestroyInstance 销毁Lite实例
func (p *Provider) DestroyInstance(ctx context.Context, instanceID string) error {
	if err := p.client.DeleteGPUInstance(ctx, instanceID); err != nil {
		return err
	}

	p.instMu.Lock()
	delete(p.instances, instanceID)
	p.instMu.Unlock()

	return nil
}

// mapPPIOStatus 将 PPIO 原始状态映射为标准小写状态
func mapPPIOStatus(status string) string {
	switch strings.ToLower(status) {
	case "pending", "creating", "starting", "pulling":
		return "creating"
	case "running":
		return "running"
	case "stopped", "exited":
		return "stopped"
	case "stopping":
		return "stopped"
	case "deleting", "destroying":
		return "destroying"
	case "deleted", "destroyed", "dead":
		return "destroyed"
	default:
		return strings.ToLower(status)
	}
}

// GetInstanceStatus 获取实例状态
func (p *Provider) GetInstanceStatus(ctx context.Context, instanceID string) (string, error) {
	inst, err := p.client.GetGPUInstance(ctx, instanceID)
	if err != nil {
		return "", err
	}

	status := mapPPIOStatus(inst.Status)
	p.instMu.Lock()
	if local, ok := p.instances[instanceID]; ok {
		local.Status = status
	}
	p.instMu.Unlock()

	return status, nil
}

// GetInstance 获取实例详情（含SSH信息）
func (p *Provider) GetInstance(ctx context.Context, instanceID string) (*provider.InstanceInfo, error) {
	log.Printf("[ppio] GetInstance called with instanceID=%q", instanceID)
	inst, err := p.client.GetGPUInstance(ctx, instanceID)
	if err != nil {
		log.Printf("[ppio] GetGPUInstance error for %q: %v", instanceID, err)
		return nil, err
	}
	log.Printf("[ppio] GetGPUInstance success for %q: status=%s", instanceID, inst.Status)

	// 解析 SSH 信息
	sshHost, sshPort, sshUser := p.parseSSHInfo(inst)
	status := mapPPIOStatus(inst.Status)

	p.instMu.Lock()
	if local, ok := p.instances[instanceID]; ok {
		local.Status = status
		local.SSHHost = sshHost
		local.SSHPort = sshPort
		local.SSHUser = sshUser
	}
	p.instMu.Unlock()

	return &provider.InstanceInfo{
		InstanceID: instanceID,
		SSHHost:    sshHost,
		SSHPort:    sshPort,
		SSHUser:    sshUser,
		Status:     status,
	}, nil
}

// parseSSHInfo 从实例详情中解析SSH连接信息
func (p *Provider) parseSSHInfo(inst *GPUInstance) (host string, port int, user string) {
	user = "root"
	port = 22

	// 1. 尝试从 sshCommand 解析，格式通常为 "ssh root@host -p port"
	if inst.SshCommand != "" {
		re := regexp.MustCompile(`ssh\s+(\w+)@([^\s]+)(?:\s+-p\s+(\d+))?`)
		if matches := re.FindStringSubmatch(inst.SshCommand); len(matches) >= 3 {
			user = matches[1]
			host = matches[2]
			if len(matches) >= 4 && matches[3] != "" {
				port, _ = strconv.Atoi(matches[3])
			}
			return
		}
	}

	// 2. 尝试从 portMappings 中找 SSH 端口（22）
	for _, pm := range inst.PortMappings {
		if pm.Type == "tcp" && (pm.Port == 22 || pm.Port == 2222) {
			port = pm.Port
			if pm.Endpoint != "" {
				host = pm.Endpoint
				return
			}
		}
	}

	// 3. 尝试从 network.ip 获取
	if inst.Network != nil && inst.Network.IP != "" {
		host = inst.Network.IP
		return
	}

	return
}
