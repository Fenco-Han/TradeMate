import type {
  AdGoal,
  ApiResponse,
  LoginResponse,
  MeResponse,
  Notification,
  ReviewSnapshot,
  Suggestion,
  Task
} from "@trademate/shared-types";

const API_BASE = "http://localhost:8080/api/v1";

export interface SuggestionPayload {
  list: Suggestion[];
  total: number;
  unread_high_risk_count: number;
}

export interface TaskPayload {
  list: Task[];
  total: number;
}

export interface NotificationPayload {
  list: Notification[];
  total: number;
}

export interface ReviewPayload {
  list: ReviewSnapshot[];
  total: number;
  status_counts: Record<string, number>;
}

export interface RunTaskItem {
  task_id: string;
  store_id: string;
  task_type: string;
  status: string;
  message: string;
  channel?: string;
  execution_mode?: string;
  attempt_count?: number;
}

export interface RunTasksOnceResult {
  picked: number;
  succeeded: number;
  failed: number;
  skipped: number;
  results: RunTaskItem[];
}

export interface ApproveSuggestionResult {
  approval_id: string;
  task_id: string;
  task_status: string;
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
  suggestions(token: string, input?: { status?: string; page_size?: number }) {
    const query = buildQuery({ status: input?.status, page_size: input?.page_size });
    return request<SuggestionPayload>(`/agents/ad/suggestions${query}`, token);
  },
  approveSuggestion(token: string, suggestionID: string) {
    return request<ApproveSuggestionResult>(`/agents/ad/suggestions/${suggestionID}/approve`, token, {
      method: "POST",
      body: JSON.stringify({ note: "approved from extension", execute_immediately: true })
    });
  },
  rejectSuggestion(token: string, suggestionID: string) {
    return request<{ suggestion_id: string; status: string }>(
      `/agents/ad/suggestions/${suggestionID}/reject`,
      token,
      {
        method: "POST",
        body: JSON.stringify({ note: "rejected from extension" })
      }
    );
  },
  tasks(token: string, input?: { status?: string; page_size?: number }) {
    const query = buildQuery({ status: input?.status, page_size: input?.page_size });
    return request<TaskPayload>(`/tasks${query}`, token);
  },
  cancelTask(token: string, taskID: string) {
    return request<Task>(`/tasks/${taskID}/cancel`, token, {
      method: "POST"
    });
  },
  retryTask(token: string, taskID: string) {
    return request<Task>(`/tasks/${taskID}/retry`, token, {
      method: "POST"
    });
  },
  runTasksOnce(token: string, input?: { limit?: number }) {
    return request<RunTasksOnceResult>("/tasks/run-once", token, {
      method: "POST",
      body: JSON.stringify({ limit: input?.limit ?? 20 })
    });
  },
  notifications(token: string, input?: { limit?: number }) {
    const query = buildQuery({ limit: input?.limit });
    return request<NotificationPayload>(`/notifications${query}`, token);
  },
  reviews(token: string, input?: { limit?: number }) {
    const query = buildQuery({ limit: input?.limit });
    return request<ReviewPayload>(`/agents/ad/reviews${query}`, token);
  },
  markNotificationRead(token: string, notificationID: string) {
    return request<{ notification_id: string; is_read: boolean }>(
      `/notifications/${notificationID}/read`,
      token,
      {
        method: "POST"
      }
    );
  }
};
