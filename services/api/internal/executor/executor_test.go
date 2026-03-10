package executor

import "testing"

func TestComparisonExecutorValidate(t *testing.T) {
	exec := newComparisonExecutor("budget_increase", "before_budget", "after_budget", true)
	if err := exec.Validate(Context{}, map[string]any{"before_budget": "10", "after_budget": "12"}); err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}

	if err := exec.Validate(Context{}, map[string]any{"before_budget": "10", "after_budget": "8"}); err == nil {
		t.Fatalf("expected error when after_budget <= before_budget")
	}
}

func TestSimpleExecutorValidate(t *testing.T) {
	exec := newSimpleExecutor("negative_keyword_add", []string{"campaign_id", "ad_group_id", "keyword_text", "match_type"}, "ok")

	payload := map[string]any{
		"campaign_id":  "cmp_001",
		"ad_group_id":  "ag_001",
		"keyword_text": "abc",
		"match_type":   "negative_exact",
	}
	if err := exec.Validate(Context{}, payload); err != nil {
		t.Fatalf("expected valid payload, got error: %v", err)
	}

	delete(payload, "keyword_text")
	if err := exec.Validate(Context{}, payload); err == nil {
		t.Fatalf("expected error when keyword_text missing")
	}
}

func TestRegistryGet(t *testing.T) {
	registry := NewDefaultRegistry()
	if _, ok := registry.Get("budget_increase"); !ok {
		t.Fatalf("budget_increase executor should exist")
	}
	if pauseExecutor, ok := registry.Get("pause_keyword"); !ok {
		t.Fatalf("pause_keyword executor should exist")
	} else if err := pauseExecutor.Validate(Context{}, map[string]any{"keyword_id": "kw_001"}); err != nil {
		t.Fatalf("pause_keyword validate failed: %v", err)
	}
	if _, ok := registry.Get("not_exists"); ok {
		t.Fatalf("unexpected executor found")
	}
}
