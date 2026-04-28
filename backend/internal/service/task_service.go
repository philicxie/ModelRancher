package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"ml-platform/internal/executor"
	"ml-platform/internal/model"
	"ml-platform/pkg/cosclient"
)

// TaskService 任务服务
type TaskService struct {
	tasks        map[string]*model.Task
	executor     *executor.Executor
	storage      *StorageService
	mu           sync.RWMutex
}

// NewTaskService 创建任务服务
func NewTaskService(exec *executor.Executor, cosClient *cosclient.Client) *TaskService {
	return &TaskService{
		tasks:    make(map[string]*model.Task),
		executor: exec,
		storage:  NewStorageService(cosClient),
	}
}

// CreateTask 创建任务
func (s *TaskService) CreateTask(ctx context.Context, req *model.CreateTaskRequest) (*model.Task, error) {
	task := &model.Task{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Status:      model.TaskStatusPending,
		Image:       req.Image,
		Command:     req.Command,
		DataPath:    req.DataPath,
		OutputPath:  req.OutputPath,
		EnvVars:     req.EnvVars,
		CreatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	// 异步执行任务
	go func() {
		execCtx := context.Background()
		if err := s.executor.ExecuteTask(execCtx, task); err != nil {
			fmt.Printf("Task execution error: %v\n", err)
		}

		// 如果有输出路径，上传到COS
		if task.OutputPath != "" && task.Status == model.TaskStatusCompleted {
			outputDir := fmt.Sprintf("/app/output/%s", task.ID)
			if err := s.storage.UploadResults(ctx, task.ID, task.OutputPath, outputDir); err != nil {
				fmt.Printf("Failed to upload results: %v\n", err)
			}
		}
	}()

	return task, nil
}

// GetTask 获取任务
func (s *TaskService) GetTask(ctx context.Context, taskID string) (*model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// ListTasks 列出所有任务
func (s *TaskService) ListTasks(ctx context.Context) ([]*model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*model.Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

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
		return fmt.Errorf("task %s is not running", taskID)
	}

	return s.executor.CancelTask(ctx, taskID)
}

// SubscribeLogs 订阅日志
func (s *TaskService) SubscribeLogs(taskID, clientID string) chan *model.TaskLog {
	return s.executor.SubscribeLogs(taskID, clientID)
}

// UnsubscribeLogs 取消订阅日志
func (s *TaskService) UnsubscribeLogs(taskID, clientID string) {
	s.executor.UnsubscribeLogs(taskID, clientID)
}