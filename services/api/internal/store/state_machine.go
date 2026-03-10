package store

import "fmt"

var suggestionTransitions = map[string]map[string]struct{}{
	"draft": {
		"ready": {},
	},
	"ready": {
		"pending_approval": {},
		"approved":         {},
		"rejected":         {},
		"expired":          {},
	},
	"pending_approval": {
		"approved": {},
		"rejected": {},
		"expired":  {},
	},
	"approved": {
		"executed": {},
		"expired":  {},
	},
}

var approvalTransitions = map[string]map[string]struct{}{
	"pending": {
		"approved":  {},
		"rejected":  {},
		"cancelled": {},
	},
}

var taskTransitions = map[string]map[string]struct{}{
	"draft": {
		"pending_approval": {},
		"approved":         {},
	},
	"pending_approval": {
		"approved":  {},
		"cancelled": {},
	},
	"approved": {
		"queued": {},
	},
	"queued": {
		"running":   {},
		"cancelled": {},
	},
	"running": {
		"succeeded": {},
		"failed":    {},
		"cancelled": {},
	},
	"failed": {
		"queued": {},
	},
}

func ValidateSuggestionTransition(from, to string) error {
	return validateTransition(suggestionTransitions, from, to, "suggestion")
}

func ValidateApprovalTransition(from, to string) error {
	return validateTransition(approvalTransitions, from, to, "approval")
}

func ValidateTaskTransition(from, to string) error {
	return validateTransition(taskTransitions, from, to, "task")
}

func validateTransition(transitions map[string]map[string]struct{}, from, to, domain string) error {
	if from == to {
		return nil
	}

	next, ok := transitions[from]
	if !ok {
		return fmt.Errorf("%s transition not allowed: %s -> %s", domain, from, to)
	}

	if _, exists := next[to]; !exists {
		return fmt.Errorf("%s transition not allowed: %s -> %s", domain, from, to)
	}

	return nil
}
