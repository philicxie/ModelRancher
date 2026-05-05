package model

import "time"

// StorageBucket 存储桶（COS中的虚拟文件夹）
type StorageBucket struct {
	ID        string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name      string    `json:"name" gorm:"type:varchar(100);not null;uniqueIndex"`
	UserID    string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	IsPublic  bool      `json:"is_public" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
}
