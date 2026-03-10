package openclaw

import (
	"context"
	"errors"
	"testing"

	"github.com/fenco/trademate/services/api/internal/config"
)

func TestRunBrowserActionDisabled(t *testing.T) {
	client := NewClient(config.Config{OpenClawFallbackEnabled: false})
	_, err := client.RunBrowserAction(context.Background(), BrowserActionRequest{
		TaskID:     "task_1",
		StoreID:    "store_1",
		ActionName: "pause_campaign",
		Payload:    map[string]any{"relay_attached": true},
	})
	if !errors.Is(err, ErrFallbackDisabled) {
		t.Fatalf("expected ErrFallbackDisabled, got %v", err)
	}
}

func TestRunBrowserActionSuccess(t *testing.T) {
	client := NewClient(config.Config{OpenClawFallbackEnabled: true})
	result, err := client.RunBrowserAction(context.Background(), BrowserActionRequest{
		TaskID:     "task_1",
		StoreID:    "store_1",
		ActionName: "pause_campaign",
		Payload:    map[string]any{"relay_attached": true},
	})
	if err != nil {
		t.Fatalf("run browser action: %v", err)
	}
	if result.Channel != "browser_fallback" || result.Status != "success" {
		t.Fatalf("unexpected result: %+v", result)
	}
}
