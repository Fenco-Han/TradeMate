package executor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Context struct {
	TaskID     string
	StoreID    string
	TargetType string
	TargetID   string
}

type ExecutionResult struct {
	ExecutionID string         `json:"execution_id"`
	Channel     string         `json:"channel"`
	Status      string         `json:"status"`
	RawResult   map[string]any `json:"raw_result"`
	Summary     string         `json:"summary"`
	FinishedAt  string         `json:"finished_at"`
}

type ActionExecutor interface {
	Name() string
	Channel() string
	Validate(ctx Context, payload map[string]any) error
	Execute(ctx Context, payload map[string]any) (ExecutionResult, error)
	Verify(ctx Context, payload map[string]any, result ExecutionResult) error
}

type Registry struct {
	executors map[string]ActionExecutor
}

func NewRegistry(executors ...ActionExecutor) *Registry {
	registry := &Registry{executors: make(map[string]ActionExecutor, len(executors))}
	for _, item := range executors {
		if item == nil {
			continue
		}
		registry.executors[item.Name()] = item
	}
	return registry
}

func NewDefaultRegistry() *Registry {
	return NewRegistry(
		newComparisonExecutor("budget_increase", "before_budget", "after_budget", true),
		newComparisonExecutor("budget_decrease", "before_budget", "after_budget", false),
		newComparisonExecutor("bid_increase", "before_bid", "after_bid", true),
		newComparisonExecutor("bid_decrease", "before_bid", "after_bid", false),
		newSimpleExecutor("campaign_pause", []string{"campaign_id"}, "campaign paused"),
		newSimpleExecutor("campaign_resume", []string{"campaign_id"}, "campaign resumed"),
		newSimpleExecutor("negative_keyword_add", []string{"campaign_id", "ad_group_id", "keyword_text", "match_type"}, "negative keyword added"),
	)
}

func (r *Registry) Get(taskType string) (ActionExecutor, bool) {
	item, ok := r.executors[taskType]
	return item, ok
}

type simpleExecutor struct {
	name         string
	requiredKeys []string
	summaryText  string
}

func newSimpleExecutor(name string, requiredKeys []string, summaryText string) ActionExecutor {
	return &simpleExecutor{name: name, requiredKeys: requiredKeys, summaryText: summaryText}
}

func (e *simpleExecutor) Name() string {
	return e.name
}

func (e *simpleExecutor) Channel() string {
	return "api"
}

func (e *simpleExecutor) Validate(_ Context, payload map[string]any) error {
	for _, key := range e.requiredKeys {
		value, ok := payload[key]
		if !ok {
			return fmt.Errorf("%s is required", key)
		}
		if strings.TrimSpace(fmt.Sprint(value)) == "" {
			return fmt.Errorf("%s is required", key)
		}
	}
	return nil
}

func (e *simpleExecutor) Execute(ctx Context, payload map[string]any) (ExecutionResult, error) {
	return buildSuccessResult(e.name, e.Channel(), ctx, payload, e.summaryText), nil
}

func (e *simpleExecutor) Verify(_ Context, _ map[string]any, result ExecutionResult) error {
	if result.Status != "success" {
		return errors.New("result status is not success")
	}
	if strings.TrimSpace(result.ExecutionID) == "" {
		return errors.New("execution_id is empty")
	}
	return nil
}

type comparisonExecutor struct {
	name      string
	beforeKey string
	afterKey  string
	increase  bool
}

func newComparisonExecutor(name, beforeKey, afterKey string, increase bool) ActionExecutor {
	return &comparisonExecutor{name: name, beforeKey: beforeKey, afterKey: afterKey, increase: increase}
}

func (e *comparisonExecutor) Name() string {
	return e.name
}

func (e *comparisonExecutor) Channel() string {
	return "api"
}

func (e *comparisonExecutor) Validate(_ Context, payload map[string]any) error {
	before, err := getFloat(payload, e.beforeKey)
	if err != nil {
		return err
	}
	after, err := getFloat(payload, e.afterKey)
	if err != nil {
		return err
	}
	if e.increase {
		if !(after > before) {
			return fmt.Errorf("%s must be greater than %s", e.afterKey, e.beforeKey)
		}
	} else {
		if !(after < before) {
			return fmt.Errorf("%s must be less than %s", e.afterKey, e.beforeKey)
		}
	}
	return nil
}

func (e *comparisonExecutor) Execute(ctx Context, payload map[string]any) (ExecutionResult, error) {
	afterValue := fmt.Sprint(payload[e.afterKey])
	summary := fmt.Sprintf("%s updated to %s", e.name, afterValue)
	return buildSuccessResult(e.name, e.Channel(), ctx, payload, summary), nil
}

func (e *comparisonExecutor) Verify(_ Context, _ map[string]any, result ExecutionResult) error {
	if result.Status != "success" {
		return errors.New("result status is not success")
	}
	if strings.TrimSpace(result.Summary) == "" {
		return errors.New("summary is empty")
	}
	return nil
}

func buildSuccessResult(name, channel string, ctx Context, payload map[string]any, summary string) ExecutionResult {
	execID := fmt.Sprintf("exec_%s_%d", ctx.TaskID, time.Now().UTC().UnixNano())
	return ExecutionResult{
		ExecutionID: execID,
		Channel:     channel,
		Status:      "success",
		RawResult: map[string]any{
			"executor_name": name,
			"task_id":       ctx.TaskID,
			"target_type":   ctx.TargetType,
			"target_id":     ctx.TargetID,
			"payload":       payload,
			"mode":          "api_executor",
			"attempt_count": 1,
		},
		Summary:    summary,
		FinishedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func getFloat(payload map[string]any, key string) (float64, error) {
	value, ok := payload[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}

	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case uint64:
		return float64(typed), nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number %s", key)
		}
		return parsed, nil
	default:
		text := strings.TrimSpace(fmt.Sprint(typed))
		parsed, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number %s", key)
		}
		return parsed, nil
	}
}
