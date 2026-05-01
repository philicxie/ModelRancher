package repository

import (
	"context"
	"fmt"
	"time"

	"ml-platform/internal/model"

	"gorm.io/gorm"
)

// InstanceRepository 实例仓储
type InstanceRepository struct {
	db *gorm.DB
}

// NewInstanceRepository 创建实例仓储
func NewInstanceRepository(db *gorm.DB) *InstanceRepository {
	return &InstanceRepository{db: db}
}

// Create 创建实例记录
func (r *InstanceRepository) Create(ctx context.Context, inst *model.Instance) error {
	if err := r.db.WithContext(ctx).Create(inst).Error; err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}
	return nil
}

// GetByID 根据ID获取实例
func (r *InstanceRepository) GetByID(ctx context.Context, id string) (*model.Instance, error) {
	var inst model.Instance
	if err := r.db.WithContext(ctx).First(&inst, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("instance not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	return &inst, nil
}

// ListByUser 列出用户的所有实例
func (r *InstanceRepository) ListByUser(ctx context.Context, userID string) ([]*model.Instance, error) {
	var instances []*model.Instance
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	return instances, nil
}

// Update 更新实例
func (r *InstanceRepository) Update(ctx context.Context, inst *model.Instance) error {
	result := r.db.WithContext(ctx).Model(&model.Instance{}).Select("*").Where("id = ?", inst.ID).Updates(inst)
	if result.Error != nil {
		return fmt.Errorf("failed to update instance: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found: %s", inst.ID)
	}
	return nil
}

// UpdateStatus 更新实例状态
func (r *InstanceRepository) UpdateStatus(ctx context.Context, id string, status model.InstanceStatus) error {
	result := r.db.WithContext(ctx).Model(&model.Instance{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("instance not found: %s", id)
	}
	return nil
}

// UpdateSSH 更新SSH信息
func (r *InstanceRepository) UpdateSSH(ctx context.Context, id string, host string, port int, user string) error {
	result := r.db.WithContext(ctx).Model(&model.Instance{}).Where("id = ?", id).Updates(map[string]interface{}{
		"ssh_host": host,
		"ssh_port": port,
		"ssh_user": user,
	})
	if result.Error != nil {
		return fmt.Errorf("failed to update ssh: %w", result.Error)
	}
	return nil
}

// Delete 删除实例记录
func (r *InstanceRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&model.Instance{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	return nil
}

// ListExpired 列出已过期的实例
func (r *InstanceRepository) ListExpired(ctx context.Context) ([]*model.Instance, error) {
	var instances []*model.Instance
	if err := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ? AND status != ?", time.Now(), model.InstanceStatusDestroyed).
		Find(&instances).Error; err != nil {
		return nil, fmt.Errorf("failed to list expired instances: %w", err)
	}
	return instances, nil
}

// UpdateStartedAt 更新启动时间
func (r *InstanceRepository) UpdateStartedAt(ctx context.Context, id string, t *time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.Instance{}).Where("id = ?", id).Update("started_at", t)
	if result.Error != nil {
		return fmt.Errorf("failed to update started_at: %w", result.Error)
	}
	return nil
}

// CountByUser 统计用户实例数量
func (r *InstanceRepository) CountByUser(ctx context.Context, userID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Instance{}).Where("user_id = ? AND status != ?", userID, model.InstanceStatusDestroyed).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
