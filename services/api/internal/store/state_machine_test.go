package store

import "testing"

func TestValidateSuggestionTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{name: "ready to approved", from: "ready", to: "approved", wantErr: false},
		{name: "pending approval to rejected", from: "pending_approval", to: "rejected", wantErr: false},
		{name: "same status is allowed", from: "approved", to: "approved", wantErr: false},
		{name: "invalid flow", from: "draft", to: "approved", wantErr: true},
		{name: "unknown from status", from: "unknown", to: "approved", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSuggestionTransition(tc.from, tc.to)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateApprovalTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{name: "pending to approved", from: "pending", to: "approved", wantErr: false},
		{name: "pending to cancelled", from: "pending", to: "cancelled", wantErr: false},
		{name: "same status is allowed", from: "pending", to: "pending", wantErr: false},
		{name: "approved to rejected is invalid", from: "approved", to: "rejected", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateApprovalTransition(tc.from, tc.to)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateTaskTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{name: "approved to queued", from: "approved", to: "queued", wantErr: false},
		{name: "queued to running", from: "queued", to: "running", wantErr: false},
		{name: "running to failed", from: "running", to: "failed", wantErr: false},
		{name: "failed to queued", from: "failed", to: "queued", wantErr: false},
		{name: "same status is allowed", from: "running", to: "running", wantErr: false},
		{name: "queued to succeeded is invalid", from: "queued", to: "succeeded", wantErr: true},
		{name: "unknown from status", from: "unknown", to: "queued", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTaskTransition(tc.from, tc.to)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
