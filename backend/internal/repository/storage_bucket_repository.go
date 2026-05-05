package repository

import (
	"context"
	"fmt"

	"ml-platform/internal/model"

	"gorm.io/gorm"
)

// StorageBucketRepository 存储桶仓储
type StorageBucketRepository struct {
	db *gorm.DB
}

// NewStorageBucketRepository 创建存储桶仓储
func NewStorageBucketRepository(db *gorm.DB) *StorageBucketRepository {
	return &StorageBucketRepository{db: db}
}

// Create 创建桶记录
func (r *StorageBucketRepository) Create(ctx context.Context, bucket *model.StorageBucket) error {
	if err := r.db.WithContext(ctx).Create(bucket).Error; err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}
	return nil
}

// GetByID 根据ID获取桶
func (r *StorageBucketRepository) GetByID(ctx context.Context, id string) (*model.StorageBucket, error) {
	var bucket model.StorageBucket
	if err := r.db.WithContext(ctx).First(&bucket, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("bucket not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}
	return &bucket, nil
}

// GetByName 根据名称获取桶
func (r *StorageBucketRepository) GetByName(ctx context.Context, name string) (*model.StorageBucket, error) {
	var bucket model.StorageBucket
	if err := r.db.WithContext(ctx).First(&bucket, "name = ?", name).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("bucket not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}
	return &bucket, nil
}

// ListByUser 列出用户的桶
func (r *StorageBucketRepository) ListByUser(ctx context.Context, userID string) ([]*model.StorageBucket, error) {
	var buckets []*model.StorageBucket
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&buckets).Error; err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	return buckets, nil
}

// ListPublic 列出公共桶
func (r *StorageBucketRepository) ListPublic(ctx context.Context) ([]*model.StorageBucket, error) {
	var buckets []*model.StorageBucket
	if err := r.db.WithContext(ctx).
		Where("is_public = ?", true).
		Order("created_at DESC").
		Find(&buckets).Error; err != nil {
		return nil, fmt.Errorf("failed to list public buckets: %w", err)
	}
	return buckets, nil
}

// Delete 删除桶记录
func (r *StorageBucketRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&model.StorageBucket{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete bucket: %w", err)
	}
	return nil
}
