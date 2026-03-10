package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fenco/trademate/services/api/internal/executor"
	"github.com/fenco/trademate/services/api/internal/models"
	"github.com/fenco/trademate/services/api/internal/store"
)

const defaultLimit = 20

const workerActorID = "system_worker"

type Repository interface {
	ListQueuedTasks(limit int, storeID string) ([]store.QueuedTask, error)
	UpdateTaskStatus(storeID, actorID, taskID, nextStatus, reason string) (models.Task, error)
	CreateNotificationForStore(storeID, messageType, priority, title, body string, targetType, targetID *string) error
}

type RunOnceInput struct {
	StoreID string
	Limit   int
	ActorID string
}

type TaskRunResult struct {
	TaskID   string `json:"task_id"`
	StoreID  string `json:"store_id"`
	TaskType string `json:"task_type"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type RunOnceResult struct {
	Picked    int             `json:"picked"`
	Succeeded int             `json:"succeeded"`
	Failed    int             `json:"failed"`
	Skipped   int             `json:"skipped"`
	Results   []TaskRunResult `json:"results"`
}

type Service struct {
	repo     Repository
	registry *executor.Registry
}

func NewService(repo Repository, registry *executor.Registry) *Service {
	if registry == nil {
		registry = executor.NewDefaultRegistry()
	}
	return &Service{repo: repo, registry: registry}
}

func (s *Service) RunOnce(ctx context.Context, input RunOnceInput) (RunOnceResult, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > 200 {
		limit = 200
	}

	actorID := strings.TrimSpace(input.ActorID)
	if actorID == "" {
		actorID = workerActorID
	}

	queuedTasks, err := s.repo.ListQueuedTasks(limit, input.StoreID)
	if err != nil {
		return RunOnceResult{}, err
	}

	result := RunOnceResult{
		Picked:  len(queuedTasks),
		Results: make([]TaskRunResult, 0, len(queuedTasks)),
	}

	for _, item := range queuedTasks {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		runningTask, claimErr := s.repo.UpdateTaskStatus(item.StoreID, actorID, item.Task.ID, "running", "picked by worker")
		if claimErr != nil {
			result.Skipped++
			result.Results = append(result.Results, TaskRunResult{
				TaskID:   item.Task.ID,
				StoreID:  item.StoreID,
				TaskType: item.Task.TaskType,
				Status:   item.Task.Status,
				Message:  claimErr.Error(),
			})
			continue
		}

		execResult, execErr := s.executeTask(ctx, item.StoreID, runningTask)
		if execErr != nil {
			failedTask, markErr := s.repo.UpdateTaskStatus(item.StoreID, actorID, runningTask.ID, "failed", truncateMessage(execErr.Error()))
			if markErr != nil {
				result.Skipped++
				result.Results = append(result.Results, TaskRunResult{
					TaskID:   runningTask.ID,
					StoreID:  item.StoreID,
					TaskType: runningTask.TaskType,
					Status:   runningTask.Status,
					Message:  fmt.Sprintf("execute failed: %v; mark failed error: %v", execErr, markErr),
				})
				continue
			}

			_ = notifyTaskStatus(s.repo, item.StoreID, failedTask.ID, failedTask.TaskType, failedTask.Status, truncateMessage(execErr.Error()))
			result.Failed++
			result.Results = append(result.Results, TaskRunResult{
				TaskID:   failedTask.ID,
				StoreID:  item.StoreID,
				TaskType: failedTask.TaskType,
				Status:   failedTask.Status,
				Message:  truncateMessage(execErr.Error()),
			})
			continue
		}

		succeededTask, markErr := s.repo.UpdateTaskStatus(item.StoreID, actorID, runningTask.ID, "succeeded", execResult.Summary)
		if markErr != nil {
			result.Skipped++
			result.Results = append(result.Results, TaskRunResult{
				TaskID:   runningTask.ID,
				StoreID:  item.StoreID,
				TaskType: runningTask.TaskType,
				Status:   runningTask.Status,
				Message:  fmt.Sprintf("mark succeeded error: %v", markErr),
			})
			continue
		}

		_ = notifyTaskStatus(s.repo, item.StoreID, succeededTask.ID, succeededTask.TaskType, succeededTask.Status, execResult.Summary)
		result.Succeeded++
		result.Results = append(result.Results, TaskRunResult{
			TaskID:   succeededTask.ID,
			StoreID:  item.StoreID,
			TaskType: succeededTask.TaskType,
			Status:   succeededTask.Status,
			Message:  execResult.Summary,
		})
	}

	return result, nil
}

func (s *Service) executeTask(_ context.Context, storeID string, task models.Task) (executor.ExecutionResult, error) {
	execHandler, exists := s.registry.Get(task.TaskType)
	if !exists {
		return executor.ExecutionResult{}, fmt.Errorf("unsupported task_type: %s", task.TaskType)
	}

	payload := map[string]any{}
	if strings.TrimSpace(task.PayloadJSON) != "" {
		if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
			return executor.ExecutionResult{}, fmt.Errorf("invalid payload_json: %w", err)
		}
	}

	execCtx := executor.Context{
		TaskID:     task.ID,
		StoreID:    storeID,
		TargetType: task.TargetType,
		TargetID:   task.TargetID,
	}

	if err := execHandler.Validate(execCtx, payload); err != nil {
		return executor.ExecutionResult{}, fmt.Errorf("validate %s failed: %w", task.TaskType, err)
	}

	execResult, err := execHandler.Execute(execCtx, payload)
	if err != nil {
		return executor.ExecutionResult{}, fmt.Errorf("execute %s failed: %w", task.TaskType, err)
	}

	if err := execHandler.Verify(execCtx, payload, execResult); err != nil {
		return executor.ExecutionResult{}, fmt.Errorf("verify %s failed: %w", task.TaskType, err)
	}

	return execResult, nil
}

func notifyTaskStatus(repo Repository, storeID, taskID, taskType, status, detail string) error {
	targetType := "task"
	targetID := taskID
	priority := "medium"
	title := "Task updated"
	body := fmt.Sprintf("Task %s status changed to %s", taskType, status)

	switch status {
	case "failed":
		priority = "high"
		title = "Task failed"
		body = fmt.Sprintf("Task %s failed: %s", taskType, detail)
	case "succeeded":
		title = "Task succeeded"
		if strings.TrimSpace(detail) != "" {
			body = fmt.Sprintf("Task %s succeeded: %s", taskType, detail)
		}
	}

	return repo.CreateNotificationForStore(storeID, "task_update", priority, title, truncateMessage(body), &targetType, &targetID)
}

func truncateMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if len(trimmed) <= 240 {
		return trimmed
	}
	return trimmed[:240]
}
