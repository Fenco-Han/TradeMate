package ads

import (
	"context"
	"testing"

	"github.com/fenco/trademate/services/api/internal/config"
)

func TestFetchPreviewDataReturnsMockWhenNotConfigured(t *testing.T) {
	client := NewClient(config.Config{})

	data, err := client.FetchPreviewData(context.Background(), "store_test")
	if err != nil {
		t.Fatalf("fetch preview data: %v", err)
	}

	if data.Source != "mock" {
		t.Fatalf("expected mock source, got %s", data.Source)
	}
	if len(data.Campaigns) == 0 || len(data.Keywords) == 0 || len(data.SearchTerms) == 0 {
		t.Fatalf("expected non-empty mock payload")
	}
}

func TestJoinErrors(t *testing.T) {
	err := joinErrors(nil, nil)
	if err != nil {
		t.Fatalf("expected nil when all errors are nil")
	}
}
