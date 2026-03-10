package worker

import (
	"errors"
	"testing"

	"github.com/fenco/trademate/services/api/internal/models"
	"github.com/fenco/trademate/services/api/internal/store"
)

type fakeRepo struct {
	queued          []store.QueuedTask
	notifications   int
	reviewSnapshots int
	listQueuedErr   error
	updateStatusErr map[string]error
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

	svc := NewService(repo, nil)
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

	svc := NewService(repo, nil)
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
}
