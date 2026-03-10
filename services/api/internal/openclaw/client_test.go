package openclaw

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	if mode, ok := result.RawResult["mode"]; !ok || mode != "openclaw_mock" {
		t.Fatalf("expected mock mode, got %+v", result.RawResult)
	}
	if attempts, ok := result.RawResult["attempt_count"]; !ok || attempts != 1 {
		t.Fatalf("expected attempt_count=1, got %+v", result.RawResult)
	}
}

func TestRunBrowserActionRuntimeSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if token := r.Header.Get("Authorization"); token != "Bearer test-token" {
			t.Fatalf("expected bearer token, got %q", token)
		}

		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if reqBody["action_name"] != "pause_campaign" {
			t.Fatalf("unexpected action_name: %+v", reqBody)
		}
		if reqBody["attached"] != true {
			t.Fatalf("expected attached=true, got %+v", reqBody["attached"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"accepted": true,
			"execution_id": "oc_runtime_1",
			"channel": "browser_fallback",
			"status": "success",
			"summary": "runtime ok",
			"raw_result": {"mode":"openclaw_runtime","check":"ok"},
			"finished_at": "2026-03-10T10:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient(config.Config{
		OpenClawFallbackEnabled: true,
		OpenClawRuntimeURL:      server.URL,
		OpenClawRuntimeToken:    "test-token",
	})
	result, err := client.RunBrowserAction(context.Background(), BrowserActionRequest{
		TaskID:     "task_1",
		StoreID:    "store_1",
		ActionName: "pause_campaign",
		Payload:    map[string]any{"relay_attached": true},
	})
	if err != nil {
		t.Fatalf("run browser action: %v", err)
	}
	if result.ExecutionID != "oc_runtime_1" {
		t.Fatalf("expected execution_id oc_runtime_1, got %s", result.ExecutionID)
	}
	if result.Summary != "runtime ok" {
		t.Fatalf("unexpected summary: %s", result.Summary)
	}
	if attempts, ok := result.RawResult["attempt_count"]; !ok || attempts != 1 {
		t.Fatalf("expected attempt_count=1, got %+v", result.RawResult)
	}
}

func TestRunBrowserActionRuntimeRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"accepted": false,
			"error_code": "UNSUPPORTED_PAGE",
			"message": "unsupported target page"
		}`))
	}))
	defer server.Close()

	client := NewClient(config.Config{
		OpenClawFallbackEnabled: true,
		OpenClawRuntimeURL:      server.URL,
	})
	_, err := client.RunBrowserAction(context.Background(), BrowserActionRequest{
		TaskID:     "task_1",
		StoreID:    "store_1",
		ActionName: "pause_campaign",
		Payload:    map[string]any{"relay_attached": true},
	})
	if !errors.Is(err, ErrRuntimeRejected) {
		t.Fatalf("expected ErrRuntimeRejected, got %v", err)
	}
	if !strings.Contains(err.Error(), "UNSUPPORTED_PAGE") {
		t.Fatalf("expected error contains code, got %v", err)
	}
}

func TestRunBrowserActionRuntimeHTTPError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "runtime unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClientWithHTTP(
		config.Config{
			OpenClawFallbackEnabled: true,
			OpenClawRuntimeURL:      server.URL,
		},
		&http.Client{Timeout: 2 * time.Second},
	)
	_, err := client.RunBrowserAction(context.Background(), BrowserActionRequest{
		TaskID:     "task_1",
		StoreID:    "store_1",
		ActionName: "pause_campaign",
		Payload:    map[string]any{"relay_attached": true},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("expected status error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected retry once for 502, got calls=%d", calls)
	}
}

func TestRunBrowserActionRuntimeRetryThenSuccess(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "temporary unavailable", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"accepted": true,
			"execution_id": "oc_retry_ok",
			"channel": "browser_fallback",
			"status": "success",
			"summary": "retry success"
		}`))
	}))
	defer server.Close()

	client := NewClient(config.Config{
		OpenClawFallbackEnabled: true,
		OpenClawRuntimeURL:      server.URL,
	})
	result, err := client.RunBrowserAction(context.Background(), BrowserActionRequest{
		TaskID:     "task_1",
		StoreID:    "store_1",
		ActionName: "pause_campaign",
		Payload:    map[string]any{"relay_attached": true},
	})
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if result.ExecutionID != "oc_retry_ok" {
		t.Fatalf("unexpected execution id: %s", result.ExecutionID)
	}
	if attempts, ok := result.RawResult["attempt_count"]; !ok || attempts != 2 {
		t.Fatalf("expected attempt_count=2, got %+v", result.RawResult)
	}
	if calls != 2 {
		t.Fatalf("expected calls=2, got %d", calls)
	}
}

func TestRunBrowserActionRuntimeNoRetryOnClientError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(config.Config{
		OpenClawFallbackEnabled: true,
		OpenClawRuntimeURL:      server.URL,
	})
	_, err := client.RunBrowserAction(context.Background(), BrowserActionRequest{
		TaskID:     "task_1",
		StoreID:    "store_1",
		ActionName: "pause_campaign",
		Payload:    map[string]any{"relay_attached": true},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("expected status 400, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected no retry for 400, got calls=%d", calls)
	}
}
