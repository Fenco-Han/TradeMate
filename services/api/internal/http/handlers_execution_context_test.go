package httpapi

import (
	"testing"

	"github.com/fenco/trademate/services/api/internal/models"
)

func TestDeriveTaskExecutionContextWithoutSnapshot(t *testing.T) {
	task := models.Task{
		Status:      "queued",
		PayloadJSON: `{"before_budget":"10","after_budget":"12"}`,
	}

	got := deriveTaskExecutionContext(task, nil)
	if got.Channel != "api" {
		t.Fatalf("expected channel api, got %s", got.Channel)
	}
	if got.Status != "pending" {
		t.Fatalf("expected status pending, got %s", got.Status)
	}
	if got.FallbackRequested {
		t.Fatalf("expected fallback_requested=false")
	}
	if got.FallbackUsed {
		t.Fatalf("expected fallback_used=false")
	}
}

func TestDeriveTaskExecutionContextFallbackPlanned(t *testing.T) {
	task := models.Task{
		Status:      "running",
		PayloadJSON: `{"force_fallback":true}`,
	}

	got := deriveTaskExecutionContext(task, nil)
	if got.Channel != "browser_fallback" {
		t.Fatalf("expected channel browser_fallback, got %s", got.Channel)
	}
	if got.Status != "pending" {
		t.Fatalf("expected status pending, got %s", got.Status)
	}
	if !got.FallbackRequested {
		t.Fatalf("expected fallback_requested=true")
	}
	if got.FallbackUsed {
		t.Fatalf("expected fallback_used=false")
	}
}

func TestDeriveTaskExecutionContextFromSnapshot(t *testing.T) {
	task := models.Task{
		Status:      "succeeded",
		PayloadJSON: `{"before_budget":"10","after_budget":"12"}`,
	}
	snapshot := &models.ReviewSnapshot{
		BeforeMetrics: map[string]any{},
		AfterMetrics: map[string]any{
			"execution_channel": "api",
			"execution_status":  "success",
			"execution_id":      "exec_1",
			"fallback_used":     false,
		},
	}

	got := deriveTaskExecutionContext(task, snapshot)
	if got.Channel != "api" {
		t.Fatalf("expected channel api, got %s", got.Channel)
	}
	if got.Status != "success" {
		t.Fatalf("expected status success, got %s", got.Status)
	}
	if got.ExecutionID == nil || *got.ExecutionID != "exec_1" {
		t.Fatalf("expected execution_id exec_1, got %v", got.ExecutionID)
	}
	if got.FallbackUsed {
		t.Fatalf("expected fallback_used=false")
	}
}

func TestDeriveTaskExecutionContextFallbackFailed(t *testing.T) {
	task := models.Task{
		Status:      "failed",
		PayloadJSON: `{"force_fallback":true}`,
	}
	snapshot := &models.ReviewSnapshot{
		BeforeMetrics: map[string]any{
			"fallback_requested": true,
		},
		AfterMetrics: map[string]any{
			"execution_status": "failed",
		},
	}

	got := deriveTaskExecutionContext(task, snapshot)
	if got.Channel != "browser_fallback" {
		t.Fatalf("expected channel browser_fallback, got %s", got.Channel)
	}
	if got.Status != "failed" {
		t.Fatalf("expected status failed, got %s", got.Status)
	}
	if !got.FallbackRequested {
		t.Fatalf("expected fallback_requested=true")
	}
	if !got.FallbackUsed {
		t.Fatalf("expected fallback_used=true")
	}
}
