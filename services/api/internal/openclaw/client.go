package openclaw

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fenco/trademate/services/api/internal/config"
)

var (
	ErrFallbackDisabled = errors.New("openclaw browser fallback is disabled")
	ErrRelayNotAttached = errors.New("openclaw relay is not attached")
)

type BrowserActionRequest struct {
	StoreID    string
	TaskID     string
	ActionName string
	Payload    map[string]any
}

type BrowserActionResult struct {
	ExecutionID string         `json:"execution_id"`
	Channel     string         `json:"channel"`
	Status      string         `json:"status"`
	Summary     string         `json:"summary"`
	RawResult   map[string]any `json:"raw_result"`
	FinishedAt  string         `json:"finished_at"`
}

type Runner interface {
	RunBrowserAction(ctx context.Context, req BrowserActionRequest) (BrowserActionResult, error)
}

type Client struct {
	enabled bool
}

func NewClient(cfg config.Config) *Client {
	return &Client{enabled: cfg.OpenClawFallbackEnabled}
}

func (c *Client) RunBrowserAction(_ context.Context, req BrowserActionRequest) (BrowserActionResult, error) {
	if !c.enabled {
		return BrowserActionResult{}, ErrFallbackDisabled
	}
	if strings.TrimSpace(req.ActionName) == "" {
		return BrowserActionResult{}, errors.New("action_name is required")
	}
	if !relayAttached(req.Payload) {
		return BrowserActionResult{}, ErrRelayNotAttached
	}

	execID := fmt.Sprintf("oc_%s_%d", req.TaskID, time.Now().UTC().UnixNano())
	summary := fmt.Sprintf("fallback action %s executed", req.ActionName)
	return BrowserActionResult{
		ExecutionID: execID,
		Channel:     "browser_fallback",
		Status:      "success",
		Summary:     summary,
		RawResult: map[string]any{
			"task_id":     req.TaskID,
			"store_id":    req.StoreID,
			"action_name": req.ActionName,
			"mode":        "openclaw_mock",
		},
		FinishedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func relayAttached(payload map[string]any) bool {
	if len(payload) == 0 {
		return false
	}
	value, exists := payload["relay_attached"]
	if !exists {
		return false
	}
	attached, ok := value.(bool)
	return ok && attached
}
