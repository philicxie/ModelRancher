package model

import (
	"time"
)

// InstanceStatus 实例状态
type InstanceStatus string

const (
	InstanceStatusCreating  InstanceStatus = "creating"
	InstanceStatusRunning   InstanceStatus = "running"
	InstanceStatusStopped   InstanceStatus = "stopped"
	InstanceStatusStarting  InstanceStatus = "starting"
	InstanceStatusDestroying InstanceStatus = "destroying"
	InstanceStatusDestroyed InstanceStatus = "destroyed"
	InstanceStatusFailed    InstanceStatus = "failed"
)

// Instance 租赁实例记录
type Instance struct {
	ID             string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	Provider       string         `json:"provider" gorm:"type:varchar(50);not null"`
	ProviderInstID string         `json:"provider_inst_id" gorm:"type:varchar(100)"`
	OfferID        string         `json:"offer_id" gorm:"type:varchar(100)"`
	UserID         string         `json:"user_id" gorm:"type:varchar(36);not null;index"`
	Name           string         `json:"name" gorm:"type:varchar(255)"`
	Image          string         `json:"image" gorm:"type:varchar(255)"`
	Status         InstanceStatus `json:"status" gorm:"type:varchar(50);not null"`
	SSHHost        string         `json:"ssh_host" gorm:"type:varchar(255)"`
	SSHPort        int            `json:"ssh_port"`
	SSHUser        string         `json:"ssh_user" gorm:"type:varchar(50)"`
	Password       string         `json:"password" gorm:"type:varchar(255)"`
	SSHCommand     string         `json:"ssh_command" gorm:"type:varchar(255)"`
	DiskSize       int            `json:"disk_size"`
	PricePerHour   float64        `json:"price_per_hour"`
	GPUName        string         `json:"gpu_name" gorm:"type:varchar(100)"`
	NumGPUs        int            `json:"num_gpus"`
	Location       string         `json:"location" gorm:"type:varchar(100)"`
	ExpiresAt      *time.Time     `json:"expires_at"`
	CreatedAt      time.Time      `json:"created_at" gorm:"not null"`
	StartedAt      *time.Time     `json:"started_at"`
	StoppedAt      *time.Time     `json:"stopped_at"`
	DestroyedAt    *time.Time     `json:"destroyed_at"`
	ErrorMsg       string         `json:"error_msg" gorm:"type:text"`
}

// WorkOrder 历史工单（实例租赁记录）
type WorkOrder struct {
	Instance
	DurationHours float64 `json:"duration_hours"`
	TotalCost     float64 `json:"total_cost"`
}

// CreateOfferRequest 创建租赁请求
type CreateOfferRequest struct {
	Provider      string  `json:"provider" binding:"required"`
	OfferID       string  `json:"offer_id" binding:"required"`
	UserID        string  `json:"user_id" binding:"required"`
	Name          string  `json:"name"`
	Image         string  `json:"image"`
	DiskSize      int     `json:"disk_size"`
	DurationHours int     `json:"duration_hours"` // 0 表示永久
	PricePerHour  float64 `json:"price_per_hour"`
	GPUName       string  `json:"gpu_name"`
	NumGPUs       int     `json:"num_gpus"`
	Location      string  `json:"location"`
}
