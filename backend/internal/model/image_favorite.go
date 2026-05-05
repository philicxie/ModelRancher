package model

import (
	"time"

	"github.com/google/uuid"
)

// ImageFavorite 用户收藏的公开镜像
type ImageFavorite struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID      string    `json:"user_id" gorm:"type:varchar(36);not null;index"`
	ImageName   string    `json:"image_name" gorm:"type:varchar(255);not null"` // 如 "pytorch/pytorch"
	Description string    `json:"description" gorm:"type:text"`
	StarCount   int       `json:"star_count"`
	IsOfficial  bool      `json:"is_official"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null"`
}

// NewImageFavorite 创建新的收藏记录
func NewImageFavorite(userID, imageName, description string, starCount int, isOfficial bool) *ImageFavorite {
	return &ImageFavorite{
		ID:          uuid.New().String(),
		UserID:      userID,
		ImageName:   imageName,
		Description: description,
		StarCount:   starCount,
		IsOfficial:  isOfficial,
		CreatedAt:   time.Now(),
	}
}
