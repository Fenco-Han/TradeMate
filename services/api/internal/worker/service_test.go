package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/fenco/trademate/services/api/internal/models"
	"github.com/fenco/trademate/services/api/internal/openclaw"
	"github.com/fenco/trademate/services/api/internal/store"
)

type fakeRepo struct {
	queued          []store.QueuedTask
	notifications   int
	reviewSnapshots int
	auditLogs       int
	listQueuedErr   error
	updateStatusErr map[string]error
}

type fakeFallbackRunner struct {
	result openclaw.BrowserActionResult
	err    error
	calls  int
}

func (f *fakeRepo) ListQueuedTasks(limit int, storeID string) ([]store.QueuedTask, error) {
	if f.listQueuedErr != nil {
		return nil, f.listQueuedErr
	}
	list := make([]store.QueuedTask, 0)
	for _, item := range f.queued {
		if storeID != "" && item.StoreID != storeID {
			continue
		}
		list = append(list, item)
		if len(list) >= limit {
			break
		}
	}
	return list, nil
}

func (f *fakeRepo) UpdateTaskStatus(storeID, _ string, taskID, nextStatus, reason string) (models.Task, error) {
	if f.updateStatusErr != nil {
		if err, ok := f.updateStatusErr[taskID+"->"+nextStatus]; ok {
			return models.Task{}, err
		}
	}
	for idx := range f.queued {
		if f.queued[idx].Task.ID == taskID && f.queued[idx].StoreID == storeID {
			f.queued[idx].Task.Status = nextStatus
			if nextStatus == "failed" {
				reasonCopy := reason
				f.queued[idx].Task.FailureReason = &reasonCopy
			}
			return f.queued[idx].Task, nil
		}
	}
	return models.Task{}, errors.New("task not found")
}

func (f *fakeRepo) CreateNotificationForStore(_ string, _, _, _, _ string, _, _ *string) error {
	f.notifications++
	return nil
}

func (f *fakeRepo) UpsertReviewSnapshot(_, _, _ string, _, _ map[string]any, _ string) (models.ReviewSnapshot, error) {
	f.reviewSnapshots++
	return models.ReviewSnapshot{}, nil
}

func (f *fakeRepo) CreateAuditLog(_, _, _, _, _, _, _ string) error {
	f.auditLogs++
	return nil
}

func (f *fakeFallbackRunner) RunBrowserAction(_ context.Context, _ openclaw.BrowserActionRequest) (openclaw.BrowserActionResult, error) {
	f.calls++
	if f.err != nil {
		return openclaw.BrowserActionResult{}, f.err
	}
	return f.result, nil
}

func TestRunOnceSuccess(t *testing.T) {
	repo := &fakeRepo{
		queued: []store.QueuedTask{{
			StoreID: "store_1",
			Task: models.Task{
				ID:          "task_1",
				TaskType:    "budget_increase",
				TargetType:  "campaign",
				TargetID:    "cmp_1",
				PayloadJSON: `{"before_budget":"10","after_budget":"12"}`,
				Status:      "queued",
			},
		}},
	}

	svc := NewService(repo, nil, nil)
	result, err := svc.RunOnce(t.Context(), RunOnceInput{Limit: 10})
	if err != nil {
		t.Fatalf("run once error: %v", err)
	}

	if result.Succeeded != 1 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Results) != 1 || result.Results[0].Status != "succeeded" {
		t.Fatalf("unexpected run details: %+v", result.Results)
	}
	if repo.notifications != 1 {
		t.Fatalf("expected 1 notification, got %d", repo.notifications)
	}
	if repo.reviewSnapshots != 1 {
		t.Fatalf("expected 1 review snapshot, got %d", repo.reviewSnapshots)
	}
	if repo.auditLogs != 1 {
		t.Fatalf("expected 1 audit log, got %d", repo.auditLogs)
	}
}

func TestRunOnceFailedForInvalidPayload(t *testing.T) {
	repo := &fakeRepo{
		queued: []store.QueuedTask{{
			StoreID: "store_1",
			Task: models.Task{
				ID:          "task_2",
				TaskType:    "budget_increase",
				TargetType:  "campaign",
				TargetID:    "cmp_1",
				PayloadJSON: `{"before_budget":"10","after_budget":"5"}`,
				Status:      "queued",
			},
		}},
	}

	svc := NewService(repo, nil, nil)
	result, err := svc.RunOnce(t.Context(), RunOnceInput{Limit: 10})
	if err != nil {
		t.Fatalf("run once error: %v", err)
	}

	if result.Failed != 1 || result.Succeeded != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Results) != 1 || result.Results[0].Status != "failed" {
		t.Fatalf("unexpected run details: %+v", result.Results)
	}
	if repo.notifications != 1 {
		t.Fatalf("expected 1 notification, got %d", repo.notifications)
	}
	if repo.reviewSnapshots != 1 {
		t.Fatalf("expected 1 review snapshot, got %d", repo.reviewSnapshots)
	}
	if repo.auditLogs != 1 {
		t.Fatalf("expected 1 audit log, got %d", repo.auditLogs)
	}
}

func TestRunOnceFallbackExecution(t *testing.T) {
	approvedBy := "u_approver"
	repo := &fakeRepo{
		queued: []store.QueuedTask{{
			StoreID: "store_1",
			Task: models.Task{
				ID:         "task_fb_1",
				TaskType:   "campaign_pause",
				TargetType: "campaign",
				TargetID:   "cmp_1",
				PayloadJSON: `{
				  "campaign_id":"cmp_1",
				  "force_fallback":true,
				  "relay_attached":true
				}`,
				Status:     "queued",
				ApprovedBy: &approvedBy,
			},
		}},
	}
	fallback := &fakeFallbackRunner{
		result: openclaw.BrowserActionResult{
			ExecutionID: "oc_1",
			Channel:     "browser_fallback",
			Status:      "success",
			Summary:     "fallback ok",
		},
	}

	svc := NewService(repo, nil, fallback)
	result, err := svc.RunOnce(t.Context(), RunOnceInput{Limit: 10})
	if err != nil {
		t.Fatalf("run once error: %v", err)
	}

	if result.Succeeded != 1 || result.Failed != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if fallback.calls != 1 {
		t.Fatalf("expected fallback runner called once, got %d", fallback.calls)
	}
	if repo.auditLogs < 2 {
		t.Fatalf("expected at least 2 audit logs, got %d", repo.auditLogs)
	}
}
