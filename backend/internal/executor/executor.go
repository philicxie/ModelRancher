package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"ml-platform/internal/model"
)

// Executor 任务执行器
type Executor struct {
	dataDir       string
	outputDir     string
	tasks         map[string]*TaskState
	mu            sync.RWMutex
	logChan       chan *model.TaskLog
	logSubs       map[string]map[string]chan *model.TaskLog
	logMu         sync.RWMutex
}

// TaskState 任务执行状态
type TaskState struct {
	Task         *model.Task
	CancelFunc   context.CancelFunc
	OutputDir    string
	DataDir      string
}

// NewExecutor 创建执行器
func NewExecutor(dataDir, outputDir string) *Executor {
	// 确保目录存在
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(outputDir, 0755)

	return &Executor{
		dataDir:  dataDir,
		outputDir: outputDir,
		tasks:    make(map[string]*TaskState),
		logChan:  make(chan *model.TaskLog, 1000),
		logSubs:  make(map[string]map[string]chan *model.TaskLog),
	}
}

// ExecuteTask 执行任务
func (e *Executor) ExecuteTask(ctx context.Context, task *model.Task) error {
	log.Printf("[Task %s] Starting execution with image: %s", task.ID, task.Image)

	// 创建任务上下文
	taskCtx, cancel := context.WithCancel(ctx)

	// 创建输出目录
	outputDir := filepath.Join(e.outputDir, task.ID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 创建数据目录
	dataDir := filepath.Join(e.dataDir, task.ID)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// 保存任务状态
	e.mu.Lock()
	e.tasks[task.ID] = &TaskState{
		Task:      task,
		CancelFunc: cancel,
		OutputDir: outputDir,
		DataDir:   dataDir,
	}
	e.mu.Unlock()

	// 记录日志
	e.emitLog(task.ID, "info", fmt.Sprintf("Task started with image: %s", task.Image))
	e.emitLog(task.ID, "info", fmt.Sprintf("Command: %s", task.Command))
	e.emitLog(task.ID, "info", fmt.Sprintf("Output directory: %s", outputDir))

	// 执行Docker命令
	cmdStr := fmt.Sprintf(
		"docker run --rm -v %s:/data -v %s:/output %s sh -c '%s'",
		dataDir, outputDir, task.Image, task.Command,
	)

	e.emitLog(task.ID, "info", fmt.Sprintf("Executing: %s", cmdStr))

	cmd := exec.CommandContext(taskCtx, "sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()

	// 记录输出
	outputLines := string(output)
	if outputLines != "" {
		for _, line := range splitLines(outputLines) {
			if line != "" {
				e.emitLog(task.ID, "info", line)
			}
		}
	}

	// 处理结果
	if err != nil {
		if ctx.Err() == context.Canceled {
			e.emitLog(task.ID, "info", "Task cancelled by user")
			task.Status = model.TaskStatusCancelled
		} else {
			e.emitLog(task.ID, "error", fmt.Sprintf("Task failed: %v", err))
			task.Status = model.TaskStatusFailed
			task.ErrorMsg = err.Error()
		}
	} else {
		e.emitLog(task.ID, "info", "Task completed successfully")
		task.Status = model.TaskStatusCompleted
	}

	// 清理状态
	e.mu.Lock()
	delete(e.tasks, task.ID)
	e.mu.Unlock()

	// 清理目录
	go func() {
		time.Sleep(5 * time.Second)
		os.RemoveAll(outputDir)
		os.RemoveAll(dataDir)
	}()

	return nil
}

// CancelTask 取消任务
func (e *Executor) CancelTask(ctx context.Context, taskID string) error {
	e.mu.RLock()
	state, ok := e.tasks[taskID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task %s not found or not running", taskID)
	}

	state.CancelFunc()
	return nil
}

// SubscribeLogs 订阅任务日志
func (e *Executor) SubscribeLogs(taskID, clientID string) chan *model.TaskLog {
	e.logMu.Lock()
	defer e.logMu.Unlock()

	if e.logSubs[taskID] == nil {
		e.logSubs[taskID] = make(map[string]chan *model.TaskLog)
	}

	ch := make(chan *model.TaskLog, 100)
	e.logSubs[taskID][clientID] = ch
	return ch
}

// UnsubscribeLogs 取消订阅日志
func (e *Executor) UnsubscribeLogs(taskID, clientID string) {
	e.logMu.Lock()
	defer e.logMu.Unlock()

	if subs, ok := e.logSubs[taskID]; ok {
		if ch, ok := subs[clientID]; ok {
			close(ch)
			delete(subs, clientID)
		}
		if len(subs) == 0 {
			delete(e.logSubs, taskID)
		}
	}
}

// EmitLog 公共方法：发送日志（供外部服务调用）
func (e *Executor) EmitLog(taskID, level, message string) {
	e.emitLog(taskID, level, message)
}

// emitLog 发送日志
func (e *Executor) emitLog(taskID, level, message string) {
	entry := &model.TaskLog{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		TaskID:    taskID,
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	// 发送到订阅者
	e.logMu.RLock()
	defer e.logMu.RUnlock()

	if subs, ok := e.logSubs[taskID]; ok {
		for _, ch := range subs {
			select {
			case ch <- entry:
			default:
				// Channel full, skip
			}
		}
	}
}

// splitLines 分割行
func splitLines(s string) []string {
	var lines []string
	for _, line := range s {
		if line == '\n' {
			lines = append(lines, "")
		}
	}
	if len(lines) == 0 {
		lines = append(lines, s)
	}
	return lines
}

// GetTaskState 获取任务状态
func (e *Executor) GetTaskState(taskID string) (*TaskState, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	state, ok := e.tasks[taskID]
	return state, ok
}

// MarshalJSON 自定义序列化
func (s *TaskState) MarshalJSON() ([]byte, error) {
	type Alias TaskState
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(s),
	})
}