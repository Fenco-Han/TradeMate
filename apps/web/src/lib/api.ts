import type {
  AdGoal,
  AuditLog,
  ApiResponse,
  LoginResponse,
  MeResponse,
  Notification,
  ReviewSnapshot,
  Suggestion,
  Task,
  RiskLevel
} from "@trademate/shared-types";

const API_BASE = "http://localhost:8080/api/v1";

export interface SuggestionsPayload {
  list: Suggestion[];
  total: number;
  unread_high_risk_count: number;
}

export interface TasksPayload {
  list: Task[];
  total: number;
}

export interface NotificationsPayload {
  list: Notification[];
  total: number;
}

export interface AuditLogsPayload {
  list: AuditLog[];
  total: number;
}

export interface ApproveSuggestionResult {
  approval_id: string;
  task_id: string;
  task_status: string;
}

export interface RunTaskItem {
  task_id: string;
  store_id: string;
  task_type: string;
  status: string;
  message: string;
}

export interface RunTasksOnceResult {
  picked: number;
  succeeded: number;
  failed: number;
  skipped: number;
  results: RunTaskItem[];
}

function getToken() {
  return localStorage.getItem("trademate.token") ?? "";
}

function buildQuery(params: Record<string, string | number | undefined>) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === "") {
      continue;
    }
    query.set(key, String(value));
  }

  const queryString = query.toString();
  return queryString ? `?${queryString}` : "";
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
  listSuggestions(input?: {
    status?: string;
    risk_level?: RiskLevel;
    page?: number;
    page_size?: number;
  }) {
    const query = buildQuery({
      status: input?.status,
      risk_level: input?.risk_level,
      page: input?.page,
      page_size: input?.page_size
    });
    return request<SuggestionsPayload>(`/agents/ad/suggestions${query}`);
  },
  approveSuggestion(suggestionID: string, input?: { note?: string; execute_immediately?: boolean }) {
    return request<ApproveSuggestionResult>(`/agents/ad/suggestions/${suggestionID}/approve`, {
      method: "POST",
      body: JSON.stringify({
        note: input?.note ?? "approved from web",
        execute_immediately: input?.execute_immediately ?? true
      })
    });
  },
  rejectSuggestion(suggestionID: string, input?: { note?: string }) {
    return request<{ suggestion_id: string; status: string }>(
      `/agents/ad/suggestions/${suggestionID}/reject`,
      {
        method: "POST",
        body: JSON.stringify({ note: input?.note ?? "rejected from web" })
      }
    );
  },
  batchApproveSuggestions(input: {
    suggestion_ids: string[];
    note?: string;
    execute_immediately?: boolean;
  }) {
    return request<{ results: ApproveSuggestionResult[]; total: number }>(
      "/agents/ad/suggestions/batch-approve",
      {
        method: "POST",
        body: JSON.stringify({
          suggestion_ids: input.suggestion_ids,
          note: input.note ?? "batch approved from web",
          execute_immediately: input.execute_immediately ?? true
        })
      }
    );
  },
  listTasks(input?: {
    status?: string;
    risk_level?: RiskLevel;
    page?: number;
    page_size?: number;
  }) {
    const query = buildQuery({
      status: input?.status,
      risk_level: input?.risk_level,
      page: input?.page,
      page_size: input?.page_size
    });
    return request<TasksPayload>(`/tasks${query}`);
  },
  cancelTask(taskID: string) {
    return request<Task>(`/tasks/${taskID}/cancel`, {
      method: "POST"
    });
  },
  retryTask(taskID: string) {
    return request<Task>(`/tasks/${taskID}/retry`, {
      method: "POST"
    });
  },
  runTasksOnce(input?: { limit?: number }) {
    return request<RunTasksOnceResult>("/tasks/run-once", {
      method: "POST",
      body: JSON.stringify({ limit: input?.limit ?? 20 })
    });
  },
  getTaskReview(taskID: string) {
    return request<ReviewSnapshot>(`/agents/ad/reviews/${taskID}`);
  },
  listNotifications(input?: { limit?: number }) {
    const query = buildQuery({
      limit: input?.limit
    });
    return request<NotificationsPayload>(`/notifications${query}`);
  },
  markNotificationRead(notificationID: string) {
    return request<{ notification_id: string; is_read: boolean }>(
      `/notifications/${notificationID}/read`,
      {
        method: "POST"
      }
    );
  },
  listAuditLogs(input?: { limit?: number }) {
    const query = buildQuery({
      limit: input?.limit
    });
    return request<AuditLogsPayload>(`/audit-logs${query}`);
  }
};
