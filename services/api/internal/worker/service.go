package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fenco/trademate/services/api/internal/executor"
	"github.com/fenco/trademate/services/api/internal/models"
	"github.com/fenco/trademate/services/api/internal/openclaw"
	"github.com/fenco/trademate/services/api/internal/store"
)

const defaultLimit = 20

const workerActorID = "system_worker"

type Repository interface {
	ListQueuedTasks(limit int, storeID string) ([]store.QueuedTask, error)
	UpdateTaskStatus(storeID, actorID, taskID, nextStatus, reason string) (models.Task, error)
	CreateNotificationForStore(storeID, messageType, priority, title, body string, targetType, targetID *string) error
	UpsertReviewSnapshot(storeID, taskID, status string, beforeMetrics, afterMetrics map[string]any, summary string) (models.ReviewSnapshot, error)
	CreateAuditLog(storeID, actorID, action, targetType, targetID, result, metadata string) error
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
	repo           Repository
	registry       *executor.Registry
	fallbackRunner openclaw.Runner
}

func NewService(repo Repository, registry *executor.Registry, fallbackRunner openclaw.Runner) *Service {
	if registry == nil {
		registry = executor.NewDefaultRegistry()
	}
	return &Service{
		repo:           repo,
		registry:       registry,
		fallbackRunner: fallbackRunner,
	}
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

			beforeMetrics, afterMetrics := extractMetricsFromTaskPayload(failedTask)
			beforeMetrics, afterMetrics = annotateExecutionMetadata(failedTask, nil, execErr, beforeMetrics, afterMetrics)
			if _, reviewErr := s.repo.UpsertReviewSnapshot(item.StoreID, failedTask.ID, "partial", beforeMetrics, afterMetrics, truncateMessage(execErr.Error())); reviewErr == nil {
				_ = s.repo.CreateAuditLog(item.StoreID, actorID, "review_generated", "task", failedTask.ID, "success", `{"review_status":"partial","task_status":"failed"}`)
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

		beforeMetrics, afterMetrics := extractMetricsFromTaskPayload(succeededTask)
		beforeMetrics, afterMetrics = annotateExecutionMetadata(succeededTask, &execResult, nil, beforeMetrics, afterMetrics)
		if _, reviewErr := s.repo.UpsertReviewSnapshot(item.StoreID, succeededTask.ID, "ready", beforeMetrics, afterMetrics, execResult.Summary); reviewErr == nil {
			_ = s.repo.CreateAuditLog(item.StoreID, actorID, "review_generated", "task", succeededTask.ID, "success", `{"review_status":"ready","task_status":"succeeded"}`)
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

func (s *Service) executeTask(ctx context.Context, storeID string, task models.Task) (executor.ExecutionResult, error) {
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

	if shouldUseFallback(payload) {
		return s.executeViaFallback(ctx, storeID, task, payload)
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

func (s *Service) executeViaFallback(ctx context.Context, storeID string, task models.Task, payload map[string]any) (executor.ExecutionResult, error) {
	if task.ApprovedBy == nil || strings.TrimSpace(*task.ApprovedBy) == "" {
		return executor.ExecutionResult{}, errors.New("fallback requires approved_by")
	}
	actionName, ok := taskTypeToFallbackAction(task.TaskType)
	if !ok {
		return executor.ExecutionResult{}, fmt.Errorf("task_type %s does not support browser fallback", task.TaskType)
	}
	if s.fallbackRunner == nil {
		return executor.ExecutionResult{}, openclaw.ErrFallbackDisabled
	}

	result, err := s.fallbackRunner.RunBrowserAction(ctx, openclaw.BrowserActionRequest{
		StoreID:    storeID,
		TaskID:     task.ID,
		ActionName: actionName,
		Payload:    payload,
	})
	if err != nil {
		_ = s.repo.CreateAuditLog(storeID, "system_worker", "task_fallback_failed", "task", task.ID, "failed", fmt.Sprintf(`{"action_name":"%s","error":"%s"}`, actionName, sanitizeJSON(err.Error())))
		targetType := "task"
		targetID := task.ID
		_ = s.repo.CreateNotificationForStore(storeID, "task_fallback", "high", "Fallback execution failed", truncateMessage(fmt.Sprintf("Task %s fallback failed: %s", task.TaskType, err.Error())), &targetType, &targetID)
		return executor.ExecutionResult{}, err
	}

	_ = s.repo.CreateAuditLog(storeID, "system_worker", "task_fallback_executed", "task", task.ID, "success", fmt.Sprintf(`{"action_name":"%s","channel":"%s"}`, actionName, result.Channel))
	targetType := "task"
	targetID := task.ID
	_ = s.repo.CreateNotificationForStore(storeID, "task_fallback", "medium", "Fallback executed", truncateMessage(fmt.Sprintf("Task %s executed by browser fallback", task.TaskType)), &targetType, &targetID)
	return executor.ExecutionResult{
		ExecutionID: result.ExecutionID,
		Channel:     result.Channel,
		Status:      result.Status,
		RawResult:   result.RawResult,
		Summary:     result.Summary,
		FinishedAt:  result.FinishedAt,
	}, nil
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

func shouldUseFallback(payload map[string]any) bool {
	value, exists := payload["force_fallback"]
	if !exists {
		return false
	}
	fallback, ok := value.(bool)
	return ok && fallback
}

func taskTypeToFallbackAction(taskType string) (string, bool) {
	switch taskType {
	case "campaign_pause":
		return "pause_campaign", true
	case "campaign_resume":
		return "resume_campaign", true
	case "negative_keyword_add":
		return "add_negative_keyword", true
	case "pause_keyword":
		return "pause_keyword", true
	default:
		return "", false
	}
}

func extractMetricsFromTaskPayload(task models.Task) (map[string]any, map[string]any) {
	beforeMetrics := map[string]any{
		"task_type":   task.TaskType,
		"target_type": task.TargetType,
		"target_id":   task.TargetID,
	}
	afterMetrics := map[string]any{
		"task_type":   task.TaskType,
		"target_type": task.TargetType,
		"target_id":   task.TargetID,
	}

	payload := map[string]any{}
	if strings.TrimSpace(task.PayloadJSON) == "" {
		return beforeMetrics, map[string]any{}
	}
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		return beforeMetrics, map[string]any{}
	}

	if fallbackRequested, ok := payload["force_fallback"].(bool); ok {
		beforeMetrics["fallback_requested"] = fallbackRequested
		afterMetrics["fallback_requested"] = fallbackRequested
	}
	if relayAttached, ok := payload["relay_attached"].(bool); ok {
		beforeMetrics["relay_attached"] = relayAttached
	}

	if nestedBefore, ok := payload["before"].(map[string]any); ok {
		for key, value := range nestedBefore {
			beforeMetrics[key] = value
		}
	}
	if nestedAfter, ok := payload["after"].(map[string]any); ok {
		for key, value := range nestedAfter {
			afterMetrics[key] = value
		}
	}

	for key, value := range payload {
		switch {
		case strings.HasPrefix(key, "before_"):
			beforeMetrics[strings.TrimPrefix(key, "before_")] = value
		case strings.HasPrefix(key, "after_"):
			afterMetrics[strings.TrimPrefix(key, "after_")] = value
		}
	}

	if len(afterMetrics) == 3 {
		return beforeMetrics, map[string]any{}
	}
	return beforeMetrics, afterMetrics
}

func annotateExecutionMetadata(task models.Task, result *executor.ExecutionResult, execErr error, beforeMetrics, afterMetrics map[string]any) (map[string]any, map[string]any) {
	if beforeMetrics == nil {
		beforeMetrics = map[string]any{}
	}
	if afterMetrics == nil {
		afterMetrics = map[string]any{}
	}

	payload := parseTaskPayload(task.PayloadJSON)
	fallbackRequested, _ := payload["force_fallback"].(bool)
	if fallbackRequested {
		beforeMetrics["planned_channel"] = "browser_fallback"
	}

	if result != nil {
		if strings.TrimSpace(result.Channel) != "" {
			afterMetrics["execution_channel"] = result.Channel
		}
		if strings.TrimSpace(result.ExecutionID) != "" {
			afterMetrics["execution_id"] = result.ExecutionID
		}
		if strings.TrimSpace(result.Status) != "" {
			afterMetrics["execution_status"] = result.Status
		}
		if strings.TrimSpace(result.FinishedAt) != "" {
			afterMetrics["execution_finished_at"] = result.FinishedAt
		}
		afterMetrics["fallback_used"] = strings.TrimSpace(result.Channel) == "browser_fallback"
		if mode, ok := result.RawResult["mode"].(string); ok && strings.TrimSpace(mode) != "" {
			afterMetrics["execution_mode"] = mode
		}
		if attemptCount, exists := result.RawResult["attempt_count"]; exists {
			afterMetrics["execution_attempt_count"] = attemptCount
		}
	}

	if execErr != nil {
		afterMetrics["execution_status"] = "failed"
		afterMetrics["failure_reason"] = truncateMessage(execErr.Error())
		if fallbackRequested {
			afterMetrics["execution_channel"] = "browser_fallback"
			afterMetrics["fallback_used"] = true
		}
	}

	return beforeMetrics, afterMetrics
}

func parseTaskPayload(payloadRaw string) map[string]any {
	payload := map[string]any{}
	if strings.TrimSpace(payloadRaw) == "" {
		return payload
	}
	if err := json.Unmarshal([]byte(payloadRaw), &payload); err != nil {
		return map[string]any{}
	}
	return payload
}

func sanitizeJSON(value string) string {
	replacer := strings.NewReplacer(`\\`, `\\\\`, `"`, `\\"`, "\n", " ", "\r", " ", "\t", " ")
	return replacer.Replace(value)
}
