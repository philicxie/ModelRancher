package repository

import (
	"context"
	"fmt"

	"ml-platform/internal/model"

	"gorm.io/gorm"
)

// ImageFavoriteRepository 镜像收藏仓储
type ImageFavoriteRepository struct {
	db *gorm.DB
}

// NewImageFavoriteRepository 创建镜像收藏仓储
func NewImageFavoriteRepository(db *gorm.DB) *ImageFavoriteRepository {
	return &ImageFavoriteRepository{db: db}
}

// Create 创建收藏记录
func (r *ImageFavoriteRepository) Create(ctx context.Context, fav *model.ImageFavorite) error {
	if err := r.db.WithContext(ctx).Create(fav).Error; err != nil {
		return fmt.Errorf("failed to create image favorite: %w", err)
	}
	return nil
}

// GetByID 根据ID获取收藏记录
func (r *ImageFavoriteRepository) GetByID(ctx context.Context, id string) (*model.ImageFavorite, error) {
	var fav model.ImageFavorite
	if err := r.db.WithContext(ctx).First(&fav, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("image favorite not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get image favorite: %w", err)
	}
	return &fav, nil
}

// GetByUserAndImage 根据用户ID和镜像名获取收藏记录
func (r *ImageFavoriteRepository) GetByUserAndImage(ctx context.Context, userID, imageName string) (*model.ImageFavorite, error) {
	var fav model.ImageFavorite
	if err := r.db.WithContext(ctx).First(&fav, "user_id = ? AND image_name = ?", userID, imageName).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("image favorite not found")
		}
		return nil, fmt.Errorf("failed to get image favorite: %w", err)
	}
	return &fav, nil
}

// ListByUser 列出用户的收藏记录
func (r *ImageFavoriteRepository) ListByUser(ctx context.Context, userID string) ([]*model.ImageFavorite, error) {
	var favorites []*model.ImageFavorite
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&favorites).Error; err != nil {
		return nil, fmt.Errorf("failed to list image favorites: %w", err)
	}
	return favorites, nil
}

// Delete 删除收藏记录
func (r *ImageFavoriteRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&model.ImageFavorite{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete image favorite: %w", err)
	}
	return nil
}

// DeleteByUserAndImage 根据用户ID和镜像名删除收藏记录
func (r *ImageFavoriteRepository) DeleteByUserAndImage(ctx context.Context, userID, imageName string) error {
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND image_name = ?", userID, imageName).
		Delete(&model.ImageFavorite{}).Error; err != nil {
		return fmt.Errorf("failed to delete image favorite: %w", err)
	}
	return nil
}
