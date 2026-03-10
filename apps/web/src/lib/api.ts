import type {
  AdGoal,
  ApiResponse,
  LoginResponse,
  MeResponse,
  Suggestion
} from "@trademate/shared-types";

const API_BASE = "http://localhost:8080/api/v1";

export interface SuggestionsPayload {
  list: Suggestion[];
  total: number;
  unread_high_risk_count: number;
}

function getToken() {
  return localStorage.getItem("trademate.token") ?? "";
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(getToken() ? { Authorization: `Bearer ${getToken()}` } : {}),
      ...(init?.headers ?? {})
    }
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || "Request failed");
  }

  const payload = (await response.json()) as ApiResponse<T>;
  return payload.data;
}

export const api = {
  login(account: string, password: string) {
    return request<LoginResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ account, password })
    });
  },
  me() {
    return request<MeResponse>("/me");
  },
  getCurrentGoal() {
    return request<AdGoal>("/agent-goals/current");
  },
  saveGoal(input: {
    goal_name: string;
    acos_target?: string | null;
    daily_budget_cap?: string | null;
    risk_profile: string;
    auto_approve_enabled: boolean;
    auto_approve_budget_delta_pct?: string | null;
    auto_approve_bid_delta_pct?: string | null;
  }) {
    return request<AdGoal>("/agent-goals/current", {
      method: "PATCH",
      body: JSON.stringify(input)
    });
  },
  listSuggestions() {
    return request<SuggestionsPayload>("/agents/ad/suggestions");
  }
};
