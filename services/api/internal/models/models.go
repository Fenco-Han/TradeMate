package models

type User struct {
	ID          string  `json:"id"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	DisplayName string  `json:"display_name"`
	Status      string  `json:"status"`
}

type Store struct {
	ID        string `json:"id"`
	SiteCode  string `json:"site_code"`
	StoreName string `json:"store_name"`
	Currency  string `json:"currency"`
	Timezone  string `json:"timezone"`
	Status    string `json:"status"`
}

type RoleAssignment struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	StoreID  string `json:"store_id"`
	RoleCode string `json:"role_code"`
}

type AdGoal struct {
	ID                        string  `json:"id"`
	AgentType                 string  `json:"agent_type"`
	StoreID                   string  `json:"store_id"`
	SiteCode                  string  `json:"site_code"`
	GoalName                  string  `json:"goal_name"`
	ACOSTarget                *string `json:"acos_target,omitempty"`
	DailyBudgetCap            *string `json:"daily_budget_cap,omitempty"`
	RiskProfile               string  `json:"risk_profile"`
	AutoApproveEnabled        bool    `json:"auto_approve_enabled"`
	AutoApproveBudgetDeltaPct *string `json:"auto_approve_budget_delta_pct,omitempty"`
	AutoApproveBidDeltaPct    *string `json:"auto_approve_bid_delta_pct,omitempty"`
	Status                    string  `json:"status"`
	EffectiveFrom             string  `json:"effective_from"`
	UpdatedBy                 string  `json:"updated_by"`
}

type Suggestion struct {
	ID                 string         `json:"id"`
	AgentType          string         `json:"agent_type"`
	StoreID            string         `json:"store_id"`
	SiteCode           string         `json:"site_code"`
	GoalID             string         `json:"goal_id"`
	TargetType         string         `json:"target_type"`
	TargetID           string         `json:"target_id"`
	SuggestionType     string         `json:"suggestion_type"`
	Title              string         `json:"title"`
	ReasonSummary      string         `json:"reason_summary"`
	RiskLevel          string         `json:"risk_level"`
	ImpactEstimateJSON map[string]any `json:"impact_estimate_json,omitempty"`
	ActionPayloadJSON  map[string]any `json:"action_payload_json"`
	Status             string         `json:"status"`
	ExpiresAt          *string        `json:"expires_at,omitempty"`
	CreatedAt          string         `json:"created_at"`
}

type Approval struct {
	ID           string  `json:"id"`
	SuggestionID string  `json:"suggestion_id"`
	StoreID      string  `json:"store_id"`
	RiskLevel    string  `json:"risk_level"`
	Status       string  `json:"status"`
	RequestedBy  string  `json:"requested_by"`
	ApprovedBy   *string `json:"approved_by,omitempty"`
	DecisionNote *string `json:"decision_note,omitempty"`
	DecidedAt    *string `json:"decided_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

type Task struct {
	ID            string  `json:"id"`
	AgentType     string  `json:"agent_type"`
	SuggestionID  string  `json:"suggestion_id"`
	ApprovalID    *string `json:"approval_id,omitempty"`
	TaskType      string  `json:"task_type"`
	TargetType    string  `json:"target_type"`
	TargetID      string  `json:"target_id"`
	RiskLevel     string  `json:"risk_level"`
	PayloadJSON   string  `json:"payload_json"`
	Status        string  `json:"status"`
	RetryCount    int     `json:"retry_count"`
	FailureReason *string `json:"failure_reason,omitempty"`
	CreatedBy     string  `json:"created_by"`
	ApprovedBy    *string `json:"approved_by,omitempty"`
	ExecutedAt    *string `json:"executed_at,omitempty"`
	FinishedAt    *string `json:"finished_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type TaskEvent struct {
	ID               string  `json:"id"`
	TaskID           string  `json:"task_id"`
	FromStatus       *string `json:"from_status,omitempty"`
	ToStatus         string  `json:"to_status"`
	EventType        string  `json:"event_type"`
	EventPayloadJSON *string `json:"event_payload_json,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

type Notification struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	AgentType   string  `json:"agent_type"`
	MessageType string  `json:"message_type"`
	Priority    string  `json:"priority"`
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	TargetType  *string `json:"target_type,omitempty"`
	TargetID    *string `json:"target_id,omitempty"`
	IsRead      bool    `json:"is_read"`
	CreatedAt   string  `json:"created_at"`
}

type AuditLog struct {
	ID           string  `json:"id"`
	AgentType    string  `json:"agent_type"`
	ActorID      string  `json:"actor_id"`
	Action       string  `json:"action"`
	TargetType   string  `json:"target_type"`
	TargetID     string  `json:"target_id"`
	Result       string  `json:"result"`
	MetadataJSON *string `json:"metadata_json,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

type ReviewSnapshot struct {
	ID            string         `json:"id"`
	AgentType     string         `json:"agent_type"`
	TaskID        string         `json:"task_id"`
	StoreID       string         `json:"store_id"`
	Status        string         `json:"status"`
	BeforeMetrics map[string]any `json:"before_metrics"`
	AfterMetrics  map[string]any `json:"after_metrics,omitempty"`
	Summary       *string        `json:"summary,omitempty"`
	GeneratedAt   string         `json:"generated_at"`
}

type MeResponse struct {
	User          User             `json:"user"`
	Roles         []RoleAssignment `json:"roles"`
	Stores        []Store          `json:"stores"`
	ActiveStoreID string           `json:"active_store_id"`
}

type LoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type UpdateGoalRequest struct {
	GoalName                  string  `json:"goal_name"`
	ACOSTarget                *string `json:"acos_target"`
	DailyBudgetCap            *string `json:"daily_budget_cap"`
	RiskProfile               string  `json:"risk_profile"`
	AutoApproveEnabled        bool    `json:"auto_approve_enabled"`
	AutoApproveBudgetDeltaPct *string `json:"auto_approve_budget_delta_pct"`
	AutoApproveBidDeltaPct    *string `json:"auto_approve_bid_delta_pct"`
}

type ApproveSuggestionRequest struct {
	Note               string `json:"note"`
	ExecuteImmediately bool   `json:"execute_immediately"`
}

type RejectSuggestionRequest struct {
	Note string `json:"note"`
}

type BatchApproveRequest struct {
	SuggestionIDs      []string `json:"suggestion_ids"`
	Note               string   `json:"note"`
	ExecuteImmediately bool     `json:"execute_immediately"`
}

type ApproveSuggestionResponse struct {
	ApprovalID string `json:"approval_id"`
	TaskID     string `json:"task_id"`
	TaskStatus string `json:"task_status"`
}

type SuggestionsPayload struct {
	List                []Suggestion `json:"list"`
	Total               int          `json:"total"`
	UnreadHighRiskCount int          `json:"unread_high_risk_count"`
}

type TaskListItem struct {
	Task         Task       `json:"task"`
	Suggestion   Suggestion `json:"suggestion"`
	Approval     *Approval  `json:"approval,omitempty"`
	ReviewStatus string     `json:"review_status"`
}

type TaskDetailResponse struct {
	Task         Task        `json:"task"`
	TaskEvents   []TaskEvent `json:"task_events"`
	AuditLogs    []AuditLog  `json:"audit_logs"`
	ReviewStatus string      `json:"review_status"`
}

type APIResponse[T any] struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
	Data      T      `json:"data"`
}
