package provider

import (
	"context"
	"fmt"
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

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	TaskID     string            // 任务唯一标识
	Name       string            // 任务名称
	Image      string            // Docker镜像
	Command    string            // 训练命令
	DataPath   string            // 数据路径
	OutputPath string            // 输出路径
	GPUType    string            // GPU类型要求
	GPUs       int               // GPU数量
	DiskSize   int               // 磁盘大小(GB)
	EnvVars    map[string]string // 环境变量
	SSHKey     string            // SSH密钥
}

// TaskInfo 任务信息
type TaskInfo struct {
	ID          string    // 任务ID
	Name        string    // 任务名称
	State       TaskState // 状态
	InstanceID  string    // 云平台实例ID
	Provider    string    // 云平台名称
	CreatedAt   string    // 创建时间
	StartedAt   string    // 开始时间
	CompletedAt string    // 完成时间
	Error       string    // 错误信息
	SSHHost     string    // SSH主机
	SSHPort     int       // SSH端口
}

// ExecutorProvider 训练任务执行Provider接口
type ExecutorProvider interface {
	// CreateTask 创建训练任务
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*TaskInfo, error)

	// GetTask 获取任务状态
	GetTask(ctx context.Context, taskID string) (*TaskInfo, error)

	// ListTasks 列出所有任务
	ListTasks(ctx context.Context) ([]*TaskInfo, error)

	// CancelTask 取消任务
	CancelTask(ctx context.Context, taskID string) error

	// UploadFile 上传文件到执行实例
	UploadFile(ctx context.Context, taskID, localPath, remotePath string) error

	// DownloadFile 从执行实例下载文件
	DownloadFile(ctx context.Context, taskID, remotePath, localPath string) error

	// ExecuteSSHCommand 执行SSH命令
	ExecuteSSHCommand(ctx context.Context, taskID, command string) (stdout, stderr string, exitCode int, err error)

	// ListAvailableResources 列出可用计算资源
	ListAvailableResources(ctx context.Context, req *ResourceRequest) ([]*Resource, error)

	// GetStats 获取统计信息
	GetStats(ctx context.Context) (*Stats, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error

	// --- Lite Instance Operations ---

	// CreateInstance 创建Lite实例
	CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*InstanceInfo, error)

	// StopInstance 停止Lite实例
	StopInstance(ctx context.Context, instanceID string) error

	// StartInstance 启动Lite实例
	StartInstance(ctx context.Context, instanceID string) error

	// DestroyInstance 销毁Lite实例
	DestroyInstance(ctx context.Context, instanceID string) error

	// GetInstanceStatus 获取实例状态
	GetInstanceStatus(ctx context.Context, instanceID string) (string, error)

	// GetInstance 获取实例详情（含SSH信息）
	GetInstance(ctx context.Context, instanceID string) (*InstanceInfo, error)

	// GetInstanceMetrics 获取实例监控指标
	GetInstanceMetrics(ctx context.Context, instanceID string, startTime, endTime int64) (*InstanceMetrics, error)
}

// ResourceRequest 资源请求
type ResourceRequest struct {
	GPUType  string  // GPU类型，如 "RTX 4090", "A100"
	MinGPUs  int     // 最小GPU数量
	MinRAM   int     // 最小GPU内存(MB)
	MaxPrice float64 // 最高价格($/小时)
	Provider string  // 指定Provider，空表示所有
}

// Resource 可用计算资源
type Resource struct {
	ID          string  // 资源ID/Offer ID
	Provider    string  // 提供商名称
	GPUType     string  // GPU类型
	NumGPUs     int     // GPU数量
	GPURAM      int     // GPU内存(MB)
	DiskSpace   float64 // 可用磁盘空间(GB)
	Reliability float64 // 可靠性评分
	Price       float64 // 价格($/小时)
	Location    string  // 位置
}

// Stats 统计信息
type Stats struct {
	TotalTasks      int            // 总任务数
	RunningTasks    int            // 运行中任务数
	CompletedTasks  int            // 已完成任务数
	FailedTasks     int            // 失败任务数
	TotalCost       float64        // 累计费用
	TasksByProvider map[string]int // 按提供商分类的任务数
}

// InstanceInfo Lite实例信息
type InstanceInfo struct {
	InstanceID string // 实例ID
	SSHHost    string // SSH主机
	SSHPort    int    // SSH端口
	SSHUser    string // SSH用户名
	Password   string // SSH密码
	SSHCommand string // 原始SSH命令（如PPIO返回的 ssh root@host -p port）
	Status     string // 状态
}

// MetricPoint 单个指标数据点
type MetricPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

// GPUInstanceMetrics 单个GPU的指标数据
type GPUInstanceMetrics struct {
	GPUID string        `json:"gpu_id"`
	Items []MetricPoint `json:"items"`
}

// InstanceMetrics 实例监控指标
type InstanceMetrics struct {
	CPUUtilization      []MetricPoint        `json:"cpu_utilization"`
	MemUtilization      []MetricPoint        `json:"mem_utilization"`
	RootDiskUtilization []MetricPoint        `json:"root_disk_utilization"`
	GPUUtilizationAvg   []MetricPoint        `json:"gpu_utilization_avg"`
	GPUUtilization      []GPUInstanceMetrics `json:"gpu_utilization"`
	GPUMemUtilizationAvg []MetricPoint       `json:"gpu_mem_utilization_avg"`
	GPUMemUtilization   []GPUInstanceMetrics `json:"gpu_mem_utilization"`
}

// CreateInstanceRequest 创建Lite实例请求
type CreateInstanceRequest struct {
	OfferID  string // 资源ID/Offer ID
	Image    string // Docker镜像
	DiskSize int    // 磁盘大小(GB)
	NumGPUs  int    // GPU数量
}

// ProviderManager Provider管理器
type ProviderManager struct {
	providers         map[string]ExecutorProvider
	availableProviders map[string]bool
	defaultProvider    string
}

// NewProviderManager 创建Provider管理器
func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers:          make(map[string]ExecutorProvider),
		availableProviders: make(map[string]bool),
	}
}

// RegisterProvider 注册Provider
func (m *ProviderManager) RegisterProvider(name string, provider ExecutorProvider) {
	m.providers[name] = provider
	if m.defaultProvider == "" {
		m.defaultProvider = name
	}
}

// MarkAvailable 标记Provider为可用
func (m *ProviderManager) MarkAvailable(name string) {
	if _, ok := m.providers[name]; ok {
		m.availableProviders[name] = true
	}
}

// IsAvailable 检查Provider是否可用
func (m *ProviderManager) IsAvailable(name string) bool {
	return m.availableProviders[name]
}

// ListAvailableProviders 列出所有可用的Provider
func (m *ProviderManager) ListAvailableProviders() []string {
	names := make([]string, 0, len(m.availableProviders))
	for name := range m.availableProviders {
		names = append(names, name)
	}
	return names
}

// GetProvider 获取指定Provider
func (m *ProviderManager) GetProvider(name string) (ExecutorProvider, bool) {
	p, ok := m.providers[name]
	return p, ok
}

// GetDefaultProvider 获取默认Provider
func (m *ProviderManager) GetDefaultProvider() ExecutorProvider {
	return m.providers[m.defaultProvider]
}

// ListProviders 列出所有注册的Provider
func (m *ProviderManager) ListProviders() []string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// SetDefaultProvider 设置默认Provider
func (m *ProviderManager) SetDefaultProvider(name string) bool {
	if _, ok := m.providers[name]; ok {
		m.defaultProvider = name
		return true
	}
	return false
}

// GetAllStats 获取所有Provider的统计信息
func (m *ProviderManager) GetAllStats(ctx context.Context) map[string]*Stats {
	allStats := make(map[string]*Stats)
	for name, p := range m.providers {
		stats, _ := p.GetStats(ctx)
		if stats != nil {
			allStats[name] = stats
		}
	}
	return allStats
}

// HealthCheckAll 检查所有Provider的健康状态
func (m *ProviderManager) HealthCheckAll(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for name, p := range m.providers {
		results[name] = p.HealthCheck(ctx)
	}
	return results
}

// ResourceInfo 资源信息
type ResourceInfo struct {
	ID          string
	Provider    string
	Type        string // "gpu", "cpu"
	GPUType     string
	NumGPUs     int
	RAM         string
	GPURAM      int
	DiskSpace   float64
	Reliability float64
	Price       float64
	Location    string
	Available   bool
}

// ListAllResources 列出所有Provider的可用资源
func (m *ProviderManager) ListAllResources(ctx context.Context, req *ResourceRequest) []ResourceInfo {
	var allResources []ResourceInfo

	for name, p := range m.providers {
		// 如果指定了 Provider，跳过不匹配的
		if req.Provider != "" && name != req.Provider {
			continue
		}

		resources, err := p.ListAvailableResources(ctx, req)
		if err != nil {
			continue
		}

		for _, r := range resources {
			allResources = append(allResources, ResourceInfo{
				ID:          r.ID,
				Provider:    name,
				Type:        "gpu",
				GPUType:     r.GPUType,
				NumGPUs:     r.NumGPUs,
				RAM:         fmt.Sprintf("%dMB", r.GPURAM),
				GPURAM:      r.GPURAM,
				DiskSpace:   r.DiskSpace,
				Reliability: r.Reliability,
				Price:       r.Price,
				Location:    r.Location,
				Available:   true,
			})
		}
	}

	return allResources
}

// CreateInstance 创建Lite实例
func (m *ProviderManager) CreateInstance(ctx context.Context, providerName string, req *CreateInstanceRequest) (*InstanceInfo, error) {
	p, ok := m.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}
	return p.CreateInstance(ctx, req)
}

// StopInstance 停止Lite实例
func (m *ProviderManager) StopInstance(ctx context.Context, providerName, instanceID string) error {
	p, ok := m.providers[providerName]
	if !ok {
		return fmt.Errorf("provider not found: %s", providerName)
	}
	return p.StopInstance(ctx, instanceID)
}

// StartInstance 启动Lite实例
func (m *ProviderManager) StartInstance(ctx context.Context, providerName, instanceID string) error {
	p, ok := m.providers[providerName]
	if !ok {
		return fmt.Errorf("provider not found: %s", providerName)
	}
	return p.StartInstance(ctx, instanceID)
}

// DestroyInstance 销毁Lite实例
func (m *ProviderManager) DestroyInstance(ctx context.Context, providerName, instanceID string) error {
	p, ok := m.providers[providerName]
	if !ok {
		return fmt.Errorf("provider not found: %s", providerName)
	}
	return p.DestroyInstance(ctx, instanceID)
}

// GetInstanceStatus 获取实例状态
func (m *ProviderManager) GetInstanceStatus(ctx context.Context, providerName, instanceID string) (string, error) {
	p, ok := m.providers[providerName]
	if !ok {
		return "", fmt.Errorf("provider not found: %s", providerName)
	}
	return p.GetInstanceStatus(ctx, instanceID)
}
