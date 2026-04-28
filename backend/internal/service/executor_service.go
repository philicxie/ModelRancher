package service

import (
	"context"

	"ml-platform/internal/executor"
	"ml-platform/internal/model"
)

// ExecutorService 执行器服务
type ExecutorService struct {
	executor *executor.Executor
	storage  *StorageService
}

// NewExecutorService 创建执行器服务（使用已创建的executor）
func NewExecutorService(exec *executor.Executor) *ExecutorService {
	return &ExecutorService{
		executor: exec,
	}
}

// ExecuteTask 执行任务
func (s *ExecutorService) ExecuteTask(ctx context.Context, task *model.Task) error {
	return s.executor.ExecuteTask(ctx, task)
}

// CancelTask 取消任务
func (s *ExecutorService) CancelTask(ctx context.Context, taskID string) error {
	return s.executor.CancelTask(ctx, taskID)
}

// SubscribeLogs 订阅日志
func (s *ExecutorService) SubscribeLogs(taskID, clientID string) chan *model.TaskLog {
	return s.executor.SubscribeLogs(taskID, clientID)
}

// UnsubscribeLogs 取消订阅日志
func (s *ExecutorService) UnsubscribeLogs(taskID, clientID string) {
	s.executor.UnsubscribeLogs(taskID, clientID)
}

// SetStorage 设置存储服务
func (s *ExecutorService) SetStorage(storage *StorageService) {
	s.storage = storage
}

// UploadResults 上传结果到COS
func (s *ExecutorService) UploadResults(ctx context.Context, taskID, outputPath, localDir string) error {
	if s.storage != nil {
		return s.storage.UploadResults(ctx, taskID, outputPath, localDir)
	}
	return nil
}