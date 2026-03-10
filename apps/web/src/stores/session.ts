import { reactive } from "vue";
import type { AdGoal, MeResponse, Suggestion } from "@trademate/shared-types";
import { api } from "../lib/api";

interface SessionState {
  token: string;
  me: MeResponse | null;
  goal: AdGoal | null;
  suggestions: Suggestion[];
  unread_high_risk_count: number;
}

export const sessionState = reactive<SessionState>({
  token: localStorage.getItem("trademate.token") ?? "",
  me: null,
  goal: null,
  suggestions: [],
  unread_high_risk_count: 0
});

export async function login(account: string, password: string) {
  const data = await api.login(account, password);
  sessionState.token = data.token;
  localStorage.setItem("trademate.token", data.token);
  await hydrateSession();
}

export async function hydrateSession() {
  if (!sessionState.token) {
    return;
  }

  sessionState.me = await api.me();
  sessionState.goal = await api.getCurrentGoal();
  const suggestions = await api.listSuggestions();
  sessionState.suggestions = suggestions.list;
  sessionState.unread_high_risk_count = suggestions.unread_high_risk_count;
}

export async function saveGoal(input: {
  goal_name: string;
  acos_target?: string | null;
  daily_budget_cap?: string | null;
  risk_profile: string;
  auto_approve_enabled: boolean;
  auto_approve_budget_delta_pct?: string | null;
  auto_approve_bid_delta_pct?: string | null;
}) {
  sessionState.goal = await api.saveGoal(input);
}
