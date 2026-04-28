package model

import "time"

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// Task 训练任务
type Task struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Image       string     `json:"image"`
	Command     string     `json:"command"`
	DataPath    string     `json:"data_path"`
	OutputPath  string     `json:"output_path"`
	EnvVars     []string   `json:"env_vars"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	ExitCode    int        `json:"exit_code,omitempty"`
	ContainerID string     `json:"container_id,omitempty"`
	ErrorMsg    string     `json:"error_msg,omitempty"`
}

// TaskLog 任务日志
type TaskLog struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Image       string   `json:"image" binding:"required"`
	Command     string   `json:"command" binding:"required"`
	DataPath    string   `json:"data_path"`
	OutputPath  string   `json:"output_path"`
	EnvVars     []string `json:"env_vars"`
}

// StorageObject 对象存储条目
type StorageObject struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	IsDir        bool      `json:"is_dir"`
}