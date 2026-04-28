package repository

import (
	"context"
	"fmt"
	"time"

	"ml-platform/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskRepository 任务仓储
type TaskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务仓储
func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create 创建任务
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

// GetByID 根据ID获取任务
func (r *TaskRepository) GetByID(ctx context.Context, id string) (*model.Task, error) {
	var task model.Task
	if err := r.db.WithContext(ctx).First(&task, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return &task, nil
}

// List 列出所有任务
func (r *TaskRepository) List(ctx context.Context, limit, offset int) ([]*model.Task, error) {
	var tasks []*model.Task
	if err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	return tasks, nil
}

// Update 更新任务
func (r *TaskRepository) Update(ctx context.Context, task *model.Task) error {
	result := r.db.WithContext(ctx).Model(&model.Task{}).Select("*").Where("id = ?", task.ID).Updates(task)
	if result.Error != nil {
		return fmt.Errorf("failed to update task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found: %s", task.ID)
	}
	return nil
}

// UpdateStatus 更新任务状态
func (r *TaskRepository) UpdateStatus(ctx context.Context, id string, status model.TaskStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}
	now := time.Now()
	switch status {
	case model.TaskStatusRunning:
		updates["started_at"] = now
	case model.TaskStatusCompleted, model.TaskStatusFailed, model.TaskStatusCancelled:
		updates["completed_at"] = now
	}
	result := r.db.WithContext(ctx).Model(&model.Task{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update task status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found: %s", id)
	}
	return nil
}

// Delete 删除任务
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.Task{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("task not found: %s", id)
	}
	return nil
}

// Count 统计任务数量
func (r *TaskRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Task{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetStats 获取任务统计
func (r *TaskRepository) GetStats(ctx context.Context) (*model.TaskStats, error) {
	stats := &model.TaskStats{}
	if err := r.db.WithContext(ctx).Model(&model.Task{}).Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	if err := r.db.WithContext(ctx).Model(&model.Task{}).Where("status = ?", model.TaskStatusPending).Count(&stats.Pending).Error; err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	if err := r.db.WithContext(ctx).Model(&model.Task{}).Where("status = ?", model.TaskStatusRunning).Count(&stats.Running).Error; err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	if err := r.db.WithContext(ctx).Model(&model.Task{}).Where("status = ?", model.TaskStatusCompleted).Count(&stats.Completed).Error; err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	if err := r.db.WithContext(ctx).Model(&model.Task{}).Where("status = ?", model.TaskStatusFailed).Count(&stats.Failed).Error; err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	return stats, nil
}

// CreateLog 创建日志记录
func (r *TaskRepository) CreateLog(ctx context.Context, log *model.TaskLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("failed to create log: %w", err)
	}
	return nil
}

// GetLogs 获取任务日志
func (r *TaskRepository) GetLogs(ctx context.Context, taskID string, limit int) ([]*model.TaskLog, error) {
	var logs []*model.TaskLog
	if err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("timestamp ASC").
		Limit(limit).
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
