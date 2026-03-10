import type {
  AdGoal,
  ApiResponse,
  LoginResponse,
  MeResponse,
  Suggestion
} from "@trademate/shared-types";

const API_BASE = "http://localhost:8080/api/v1";

export interface SuggestionPayload {
  list: Suggestion[];
  total: number;
  unread_high_risk_count: number;
}

async function request<T>(path: string, token?: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers ?? {})
    }
  });

  if (!response.ok) {
    throw new Error(await response.text());
  }

  const payload = (await response.json()) as ApiResponse<T>;
  return payload.data;
}

export const extensionApi = {
  login(account: string, password: string) {
    return request<LoginResponse>("/auth/login", undefined, {
      method: "POST",
      body: JSON.stringify({ account, password })
    });
  },
  me(token: string) {
    return request<MeResponse>("/me", token);
  },
  goal(token: string) {
    return request<AdGoal>("/agent-goals/current", token);
  },
  suggestions(token: string) {
    return request<SuggestionPayload>("/agents/ad/suggestions", token);
  }
};
