export type RoleCode = "owner" | "operator" | "approver" | "viewer";
export type RiskLevel = "low" | "medium" | "high";
export type SuggestionStatus =
  | "draft"
  | "ready"
  | "pending_approval"
  | "approved"
  | "rejected"
  | "expired"
  | "executed";

export type TaskStatus =
  | "draft"
  | "pending_approval"
  | "approved"
  | "queued"
  | "running"
  | "succeeded"
  | "failed"
  | "cancelled";

export interface User {
  id: string;
  email?: string | null;
  phone?: string | null;
  display_name: string;
  status: "active" | "disabled";
}

export interface Store {
  id: string;
  site_code: string;
  store_name: string;
  currency: string;
  timezone: string;
  status: "active" | "paused";
}

export interface RoleAssignment {
  id: string;
  user_id: string;
  store_id: string;
  role_code: RoleCode;
}

export interface AdGoal {
  id: string;
  agent_type: "ad_agent";
  store_id: string;
  site_code: string;
  goal_name: string;
  acos_target?: string | null;
  daily_budget_cap?: string | null;
  risk_profile: "conservative" | "balanced" | "aggressive";
  auto_approve_enabled: boolean;
  auto_approve_budget_delta_pct?: string | null;
  auto_approve_bid_delta_pct?: string | null;
  status: "active" | "paused";
  effective_from: string;
  updated_by: string;
}

export interface Suggestion {
  id: string;
  agent_type: "ad_agent";
  store_id: string;
  site_code: string;
  goal_id: string;
  target_type: "campaign" | "ad_group" | "keyword" | "search_term";
  target_id: string;
  suggestion_type: string;
  title: string;
  reason_summary: string;
  risk_level: RiskLevel;
  impact_estimate_json?: Record<string, unknown> | null;
  action_payload_json: Record<string, unknown>;
  status: SuggestionStatus;
  expires_at?: string | null;
  created_at: string;
}

export interface Task {
  id: string;
  agent_type: string;
  suggestion_id: string;
  approval_id?: string | null;
  task_type: string;
  target_type: string;
  target_id: string;
  risk_level: RiskLevel;
  payload_json: string;
  status: TaskStatus;
  retry_count: number;
  failure_reason?: string | null;
  created_by: string;
  approved_by?: string | null;
  executed_at?: string | null;
  finished_at?: string | null;
  created_at: string;
}

export interface Notification {
  id: string;
  user_id: string;
  agent_type: string;
  message_type: string;
  priority: "low" | "medium" | "high";
  title: string;
  body: string;
  target_type?: string | null;
  target_id?: string | null;
  is_read: boolean;
  created_at: string;
}

export interface AuditLog {
  id: string;
  agent_type: string;
  actor_id: string;
  action: string;
  target_type: string;
  target_id: string;
  result: string;
  metadata_json?: string | null;
  created_at: string;
}

export interface ReviewSnapshot {
  id?: string;
  agent_type: string;
  task_id: string;
  store_id: string;
  status: "pending" | "partial" | "ready";
  before_metrics: Record<string, unknown>;
  after_metrics?: Record<string, unknown>;
  summary?: string | null;
  generated_at?: string;
}

export interface MeResponse {
  user: User;
  roles: RoleAssignment[];
  stores: Store[];
  active_store_id: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface ApiResponse<T> {
  code: string;
  message: string;
  request_id: string;
  data: T;
}
