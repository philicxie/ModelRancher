package model

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskPhase 任务阶段（用于远程执行流程）
type TaskPhase string

const (
	TaskPhasePending         TaskPhase = "pending"          // 等待启动
	TaskPhaseCreatingInstance TaskPhase = "creating_instance" // 创建实例中
	TaskPhaseInstanceReady   TaskPhase = "instance_ready"    // 实例就绪
	TaskPhaseDownloading     TaskPhase = "downloading"       // 下载输入数据中
	TaskPhaseTraining        TaskPhase = "training"          // 训练中
	TaskPhaseUploading       TaskPhase = "uploading"         // 上传输出数据中
	TaskPhaseCompleted       TaskPhase = "completed"         // 已完成
	TaskPhaseFailed          TaskPhase = "failed"            // 失败
	TaskPhaseCancelled       TaskPhase = "cancelled"         // 已取消
)

// Task 训练任务
type Task struct {
	ID           string      `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name         string      `json:"name" gorm:"type:varchar(255);not null"`
	Description  string      `json:"description" gorm:"type:text"`
	Status       TaskStatus  `json:"status" gorm:"type:varchar(50);not null"`
	Phase        TaskPhase   `json:"phase" gorm:"type:varchar(50);default:'pending'"`
	Image        string      `json:"image" gorm:"type:varchar(255);not null"`
	Command      string      `json:"command" gorm:"type:text;not null"`
	DataPath        string           `json:"data_path" gorm:"type:varchar(255)"`    // COS 输入路径 bucket/prefix
	OutputPath      string           `json:"output_path" gorm:"type:varchar(255)"`  // COS 输出路径 bucket/prefix
	EnvVars         StringSlice      `json:"env_vars" gorm:"type:jsonb"`
	StorageBindings StorageBindingList `json:"storage_bindings" gorm:"type:jsonb"`
	CreatedAt    time.Time   `json:"created_at" gorm:"not null"`
	StartedAt    *time.Time  `json:"started_at,omitempty"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
	ExitCode     int         `json:"exit_code,omitempty"`
	ContainerID  string      `json:"container_id,omitempty" gorm:"type:varchar(255)"`
	ErrorMsg     string      `json:"error_msg,omitempty" gorm:"type:text"`
	UserID       *uuid.UUID  `json:"user_id,omitempty" gorm:"type:uuid"`

	// 实例关联（远程执行时使用）
	InstanceID   string  `json:"instance_id" gorm:"type:varchar(36)"`
	Provider     string  `json:"provider" gorm:"type:varchar(50)"`
	SSHHost      string  `json:"ssh_host" gorm:"type:varchar(255)"`
	SSHPort      int     `json:"ssh_port"`
	SSHUser      string  `json:"ssh_user" gorm:"type:varchar(100)"`
	SSHPassword  string  `json:"-" gorm:"type:varchar(255)"` // 不返回给前端
}

// NewTask 创建新任务
func NewTask(name, image, command string) *Task {
	return &Task{
		ID:        uuid.New().String(),
		Name:      name,
		Image:     image,
		Command:   command,
		Status:    TaskStatusPending,
		CreatedAt: time.Now(),
	}
}

// TaskStats 任务统计
type TaskStats struct {
	Total     int64 `json:"total"`
	Pending   int64 `json:"pending"`
	Running   int64 `json:"running"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
}

// TaskLog 任务日志
type TaskLog struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	TaskID    string    `json:"task_id" gorm:"type:varchar(36);not null;index"`
	Timestamp time.Time `json:"timestamp" gorm:"not null"`
	Level     string    `json:"level" gorm:"type:varchar(50);not null"`
	Message   string    `json:"message" gorm:"type:text;not null"`
}

// StorageBinding 存储路径映射
type StorageBinding struct {
	Type     string `json:"type"`      // input 或 output
	EnvName  string `json:"env_name"`  // 环境变量名
	Path     string `json:"path"`      // COS 路径 bucket/prefix
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	Image         string   `json:"image" binding:"required"`
	Command       string   `json:"command" binding:"required"`
	DataPath      string           `json:"data_path"`           // COS 输入路径，格式：bucket/prefix
	OutputPath    string           `json:"output_path"`         // COS 输出路径，格式：bucket/prefix
	EnvVars       []string         `json:"env_vars"`
	StorageBindings []StorageBinding `json:"storage_bindings" gorm:"type:jsonb"`

	// 实例配置（如指定 provider 则创建远程实例，否则本地执行）
	Provider      string  `json:"provider"`       // vastai, ppio, local, 空表示本地
	OfferID       string  `json:"offer_id"`       // 算力市场选择的实例/产品 ID
	GPUName       string  `json:"gpu_name"`
	NumGPUs       int     `json:"num_gpus"`
	DiskSize      int     `json:"disk_size"`
	DurationHours int     `json:"duration_hours"`
	PricePerHour  float64 `json:"price_per_hour"`
}

// StorageObject 对象存储条目
type StorageObject struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	IsDir        bool      `json:"is_dir"`
}
