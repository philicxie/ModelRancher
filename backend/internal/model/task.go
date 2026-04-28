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

// Task 训练任务
type Task struct {
	ID          string      `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name        string      `json:"name" gorm:"type:varchar(255);not null"`
	Description string      `json:"description" gorm:"type:text"`
	Status      TaskStatus  `json:"status" gorm:"type:varchar(50);not null"`
	Image       string      `json:"image" gorm:"type:varchar(255);not null"`
	Command     string      `json:"command" gorm:"type:text;not null"`
	DataPath    string      `json:"data_path" gorm:"type:varchar(255)"`
	OutputPath  string      `json:"output_path" gorm:"type:varchar(255)"`
	EnvVars     StringSlice `json:"env_vars" gorm:"type:jsonb"`
	CreatedAt   time.Time   `json:"created_at" gorm:"not null"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	ExitCode    int         `json:"exit_code,omitempty"`
	ContainerID string      `json:"container_id,omitempty" gorm:"type:varchar(255)"`
	ErrorMsg    string      `json:"error_msg,omitempty" gorm:"type:text"`
	UserID      *uuid.UUID  `json:"user_id,omitempty" gorm:"type:uuid"`
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
