package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"ml-platform/internal/executor"
	"ml-platform/internal/model"
	"ml-platform/internal/repository"
	"ml-platform/pkg/cosclient"
)

// TaskService 任务服务
type TaskService struct {
	repo     *repository.TaskRepository
	executor *executor.Executor
	storage  *StorageService
	tasks    map[string]*model.Task // 本地缓存，用于快速访问
	mu       sync.RWMutex
}

// NewTaskService 创建任务服务
func NewTaskService(repo *repository.TaskRepository, exec *executor.Executor, cosClient *cosclient.Client) *TaskService {
	return &TaskService{
		repo:     repo,
		executor: exec,
		storage:  NewStorageService(cosClient),
		tasks:    make(map[string]*model.Task),
	}
}

// CreateTask 创建任务
func (s *TaskService) CreateTask(ctx context.Context, req *model.CreateTaskRequest) (*model.Task, error) {
	task := model.NewTask(req.Name, req.Image, req.Command)
	task.Description = req.Description
	task.DataPath = req.DataPath
	task.OutputPath = req.OutputPath
	task.EnvVars = model.StringSlice(req.EnvVars)

	// 保存到数据库
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task in database: %w", err)
	}

	// 更新本地缓存
	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	// 异步执行任务
	go s.executeTask(task)

	return task, nil
}

// executeTask 异步执行任务
func (s *TaskService) executeTask(task *model.Task) {
	ctx := context.Background()

	// 更新状态为运行中
	task.Status = model.TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now

	if err := s.repo.Update(ctx, task); err != nil {
		log.Printf("Failed to update task status: %v", err)
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	// 执行任务
	if err := s.executor.ExecuteTask(ctx, task); err != nil {
		task.Status = model.TaskStatusFailed
		task.ErrorMsg = err.Error()
		log.Printf("Task %s execution error: %v", task.ID, err)
	} else {
		task.Status = model.TaskStatusCompleted
		log.Printf("Task %s completed successfully", task.ID)
	}

	// 更新完成时间
	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if err := s.repo.Update(ctx, task); err != nil {
		log.Printf("Failed to update task completion: %v", err)
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	// 如果有输出路径，上传到COS
	if task.OutputPath != "" && task.Status == model.TaskStatusCompleted {
		outputDir := fmt.Sprintf("/app/output/%s", task.ID)
		if err := s.storage.UploadResults(ctx, task.ID, task.OutputPath, outputDir); err != nil {
			log.Printf("Failed to upload results for task %s: %v", task.ID, err)
		}
	}
}

// GetTask 获取任务
func (s *TaskService) GetTask(ctx context.Context, taskID string) (*model.Task, error) {
	// 先从本地缓存获取
	s.mu.RLock()
	task, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if ok {
		return task, nil
	}

	// 从数据库获取
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// 更新本地缓存
	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	return task, nil
}

// ListTasks 列出所有任务
func (s *TaskService) ListTasks(ctx context.Context) ([]*model.Task, error) {
	// 默认返回100条记录
	tasks, err := s.repo.List(ctx, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// 更新本地缓存
	s.mu.Lock()
	for _, task := range tasks {
		s.tasks[task.ID] = task
	}
	s.mu.Unlock()

	return tasks, nil
}

// CancelTask 取消任务
func (s *TaskService) CancelTask(ctx context.Context, taskID string) error {
	s.mu.RLock()
	task, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status != model.TaskStatusPending && task.Status != model.TaskStatusRunning {
		return fmt.Errorf("task %s is not in a cancellable state", taskID)
	}

	if err := s.executor.CancelTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	// 更新状态
	task.Status = model.TaskStatusCancelled
	now := time.Now()
	task.CompletedAt = &now

	if err := s.repo.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	return nil
}

// GetTaskStats 获取任务统计
func (s *TaskService) GetTaskStats(ctx context.Context) (*model.TaskStats, error) {
	return s.repo.GetStats(ctx)
}

// SubscribeLogs 订阅日志
func (s *TaskService) SubscribeLogs(taskID, clientID string) chan *model.TaskLog {
	return s.executor.SubscribeLogs(taskID, clientID)
}

// UnsubscribeLogs 取消订阅日志
func (s *TaskService) UnsubscribeLogs(taskID, clientID string) {
	s.executor.UnsubscribeLogs(taskID, clientID)
}
