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
	snapshotWrites  []snapshotWrite
}

type fakeFallbackRunner struct {
	result openclaw.BrowserActionResult
	err    error
	calls  int
}

type snapshotWrite struct {
	storeID string
	taskID  string
	status  string
	before  map[string]any
	after   map[string]any
	summary string
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

func (f *fakeRepo) UpsertReviewSnapshot(storeID, taskID, status string, before, after map[string]any, summary string) (models.ReviewSnapshot, error) {
	f.reviewSnapshots++
	f.snapshotWrites = append(f.snapshotWrites, snapshotWrite{
		storeID: storeID,
		taskID:  taskID,
		status:  status,
		before:  cloneMap(before),
		after:   cloneMap(after),
		summary: summary,
	})
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

func cloneMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
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
	if len(repo.snapshotWrites) != 1 {
		t.Fatalf("expected 1 snapshot write, got %d", len(repo.snapshotWrites))
	}
	if got := repo.snapshotWrites[0].after["execution_channel"]; got != "api" {
		t.Fatalf("expected execution_channel=api, got %v", got)
	}
	if got := repo.snapshotWrites[0].after["execution_status"]; got != "success" {
		t.Fatalf("expected execution_status=success, got %v", got)
	}
	if got := repo.snapshotWrites[0].after["fallback_used"]; got != false {
		t.Fatalf("expected fallback_used=false, got %v", got)
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
	if len(repo.snapshotWrites) != 1 {
		t.Fatalf("expected 1 snapshot write, got %d", len(repo.snapshotWrites))
	}
	if got := repo.snapshotWrites[0].status; got != "partial" {
		t.Fatalf("expected review status partial, got %s", got)
	}
	if got := repo.snapshotWrites[0].after["execution_status"]; got != "failed" {
		t.Fatalf("expected execution_status=failed, got %v", got)
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
	if len(repo.snapshotWrites) != 1 {
		t.Fatalf("expected 1 snapshot write, got %d", len(repo.snapshotWrites))
	}
	if got := repo.snapshotWrites[0].before["fallback_requested"]; got != true {
		t.Fatalf("expected fallback_requested=true, got %v", got)
	}
	if got := repo.snapshotWrites[0].after["execution_channel"]; got != "browser_fallback" {
		t.Fatalf("expected execution_channel=browser_fallback, got %v", got)
	}
	if got := repo.snapshotWrites[0].after["fallback_used"]; got != true {
		t.Fatalf("expected fallback_used=true, got %v", got)
	}
	if repo.auditLogs < 2 {
		t.Fatalf("expected at least 2 audit logs, got %d", repo.auditLogs)
	}
	if repo.notifications < 2 {
		t.Fatalf("expected at least 2 notifications, got %d", repo.notifications)
	}
}
