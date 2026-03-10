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
