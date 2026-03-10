package openclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fenco/trademate/services/api/internal/config"
)

var (
	ErrFallbackDisabled = errors.New("openclaw browser fallback is disabled")
	ErrRelayNotAttached = errors.New("openclaw relay is not attached")
	ErrRuntimeRejected  = errors.New("openclaw runtime rejected action")
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
	enabled      bool
	runtimeURL   string
	runtimeToken string
	httpClient   *http.Client
}

func NewClient(cfg config.Config) *Client {
	return NewClientWithHTTP(cfg, &http.Client{Timeout: 8 * time.Second})
}

func NewClientWithHTTP(cfg config.Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}

	return &Client{
		enabled:      cfg.OpenClawFallbackEnabled,
		runtimeURL:   strings.TrimSpace(cfg.OpenClawRuntimeURL),
		runtimeToken: strings.TrimSpace(cfg.OpenClawRuntimeToken),
		httpClient:   httpClient,
	}
}

func (c *Client) RunBrowserAction(ctx context.Context, req BrowserActionRequest) (BrowserActionResult, error) {
	if !c.enabled {
		return BrowserActionResult{}, ErrFallbackDisabled
	}
	if strings.TrimSpace(req.ActionName) == "" {
		return BrowserActionResult{}, errors.New("action_name is required")
	}
	if !relayAttached(req.Payload) {
		return BrowserActionResult{}, ErrRelayNotAttached
	}

	if c.runtimeURL != "" {
		return c.runViaRuntime(ctx, req)
	}

	return runMock(req), nil
}

func (c *Client) runViaRuntime(ctx context.Context, req BrowserActionRequest) (BrowserActionResult, error) {
	bodyRaw, err := json.Marshal(buildRuntimeRequest(req))
	if err != nil {
		return BrowserActionResult{}, fmt.Errorf("marshal runtime request: %w", err)
	}

	lastErr := error(nil)
	for attempt := 1; attempt <= 2; attempt++ {
		result, retryable, callErr := c.callRuntimeOnce(ctx, req, bodyRaw)
		if callErr == nil {
			if result.RawResult == nil {
				result.RawResult = map[string]any{}
			}
			result.RawResult["attempt_count"] = attempt
			return result, nil
		}
		lastErr = callErr

		if !retryable || attempt == 2 {
			break
		}
	}

	return BrowserActionResult{}, lastErr
}

func (c *Client) callRuntimeOnce(ctx context.Context, req BrowserActionRequest, bodyRaw []byte) (BrowserActionResult, bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.runtimeURL, bytes.NewReader(bodyRaw))
	if err != nil {
		return BrowserActionResult{}, false, fmt.Errorf("build runtime request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.runtimeToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.runtimeToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return BrowserActionResult{}, false, ctx.Err()
		}
		return BrowserActionResult{}, true, fmt.Errorf("call openclaw runtime: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return BrowserActionResult{}, false, fmt.Errorf("read runtime response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		retryable := resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests
		return BrowserActionResult{}, retryable, fmt.Errorf("openclaw runtime status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	result, parseErr := parseRuntimeResult(req, respBody)
	if parseErr != nil {
		return BrowserActionResult{}, false, parseErr
	}
	return result, false, nil
}

func runMock(req BrowserActionRequest) BrowserActionResult {
	execID := fmt.Sprintf("oc_%s_%d", req.TaskID, time.Now().UTC().UnixNano())
	summary := fmt.Sprintf("fallback action %s executed", req.ActionName)
	return BrowserActionResult{
		ExecutionID: execID,
		Channel:     "browser_fallback",
		Status:      "success",
		Summary:     summary,
		RawResult: map[string]any{
			"task_id":       req.TaskID,
			"store_id":      req.StoreID,
			"action_name":   req.ActionName,
			"mode":          "openclaw_mock",
			"attempt_count": 1,
		},
		FinishedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func buildRuntimeRequest(req BrowserActionRequest) map[string]any {
	payload := map[string]any{}
	for key, value := range req.Payload {
		payload[key] = value
	}

	request := map[string]any{
		"action_name": req.ActionName,
		"payload":     payload,
		"attached":    true,
		"store_id":    req.StoreID,
		"task_id":     req.TaskID,
	}

	if tabID, exists := req.Payload["tab_id"]; exists {
		request["tab_id"] = tabID
	}
	if pageURL, exists := req.Payload["url"]; exists {
		request["url"] = pageURL
	}
	if browser, exists := req.Payload["browser"]; exists {
		request["browser"] = browser
	}

	return request
}

func parseRuntimeResult(req BrowserActionRequest, body []byte) (BrowserActionResult, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return BrowserActionResult{}, fmt.Errorf("invalid runtime response: %w", err)
	}

	if accepted, ok := payload["accepted"].(bool); ok && !accepted {
		message := strings.TrimSpace(fmt.Sprint(payload["message"]))
		if message == "" || message == "<nil>" {
			message = "runtime rejected action"
		}
		errorCode := strings.TrimSpace(fmt.Sprint(payload["error_code"]))
		if errorCode != "" && errorCode != "<nil>" {
			return BrowserActionResult{}, fmt.Errorf("%w: %s (%s)", ErrRuntimeRejected, message, errorCode)
		}
		return BrowserActionResult{}, fmt.Errorf("%w: %s", ErrRuntimeRejected, message)
	}

	executionID := readString(payload, "execution_id")
	if executionID == "" {
		executionID = fmt.Sprintf("oc_%s_%d", req.TaskID, time.Now().UTC().UnixNano())
	}
	channel := readString(payload, "channel")
	if channel == "" {
		channel = "browser_fallback"
	}
	status := readString(payload, "status")
	if status == "" {
		status = "success"
	}
	summary := readString(payload, "summary")
	if summary == "" {
		summary = fmt.Sprintf("fallback action %s executed", req.ActionName)
	}
	finishedAt := readString(payload, "finished_at")
	if finishedAt == "" {
		finishedAt = time.Now().UTC().Format(time.RFC3339)
	}

	rawResult := map[string]any{
		"task_id":     req.TaskID,
		"store_id":    req.StoreID,
		"action_name": req.ActionName,
		"mode":        "openclaw_runtime",
	}
	if parsedRaw, ok := payload["raw_result"].(map[string]any); ok && len(parsedRaw) > 0 {
		rawResult = parsedRaw
	}

	return BrowserActionResult{
		ExecutionID: executionID,
		Channel:     channel,
		Status:      status,
		Summary:     summary,
		RawResult:   rawResult,
		FinishedAt:  finishedAt,
	}, nil
}

func readString(payload map[string]any, key string) string {
	value, exists := payload[key]
	if !exists {
		return ""
	}
	result := strings.TrimSpace(fmt.Sprint(value))
	if result == "<nil>" {
		return ""
	}
	return result
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
