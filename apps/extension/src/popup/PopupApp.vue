<template>
  <div class="popup-shell">
    <div v-if="!token" class="card">
      <h1>TradeMate</h1>
      <p>Login to view Ad Agent suggestions.</p>
      <label>
        Account
        <input v-model="account" />
      </label>
      <label>
        Password
        <input v-model="password" type="password" />
      </label>
      <button @click="submit">Login</button>
      <p class="meta">demo@trademate.dev / demo123</p>
      <p v-if="error" class="error">{{ error }}</p>
    </div>

    <div v-else class="popup-content">
      <section class="card">
        <div class="summary-grid">
          <div class="metric">
            <span>Suggestions</span>
            <strong>{{ displaySuggestions.length }}</strong>
          </div>
          <div class="metric">
            <span>Tasks</span>
            <strong>{{ tasks.length }}</strong>
          </div>
          <div class="metric">
            <span>Unread</span>
            <strong>{{ settings.reminders_enabled ? unreadNotifications : 0 }}</strong>
          </div>
          <div class="metric">
            <span>High Risk</span>
            <strong>{{ highRiskCount }}</strong>
          </div>
        </div>
      </section>

      <section class="card">
        <h2>{{ storeName }}</h2>
        <p>{{ goalSummary }}</p>
        <p class="meta">默认站点: {{ settings.default_site_code }} · 提醒: {{ settings.reminders_enabled ? "开启" : "关闭" }}</p>
      </section>

      <section class="card tabs-card">
        <div class="tabs">
          <button :class="['tab-btn', { active: activeTab === 'overview' }]" @click="activeTab = 'overview'">
            Overview
          </button>
          <button
            :class="['tab-btn', { active: activeTab === 'suggestions' }]"
            @click="activeTab = 'suggestions'"
          >
            Suggestions
          </button>
          <button :class="['tab-btn', { active: activeTab === 'tasks' }]" @click="activeTab = 'tasks'">
            Tasks
          </button>
          <button
            :class="['tab-btn', { active: activeTab === 'notifications' }]"
            @click="activeTab = 'notifications'"
          >
            Notifications
          </button>
        </div>

        <p v-if="error" class="error">{{ error }}</p>
        <p v-if="message" class="meta">{{ message }}</p>

        <div v-if="activeTab === 'overview'" class="tab-pane">
          <h3>Top Suggestions</h3>
          <ul class="popup-list">
            <li v-for="item in topSuggestions" :key="item.id">
              <div class="row">
                <strong>{{ item.title }}</strong>
                <span :class="['risk-pill', item.risk_level]">{{ item.risk_level }}</span>
              </div>
              <p>{{ item.reason_summary }}</p>
              <p v-if="settings.show_impact_estimate && hasImpactEstimate(item)" class="meta">
                影响预估: {{ summarizeImpact(item) }}
              </p>
            </li>
          </ul>
        </div>

        <div v-else-if="activeTab === 'suggestions'" class="tab-pane">
          <div class="row">
            <h3>Pending Suggestions</h3>
            <button class="small" :disabled="loading" @click="refreshAll">Refresh</button>
          </div>
          <p v-if="settings.show_high_risk_only" class="meta">当前仅展示高风险建议</p>
          <ul class="popup-list">
            <li v-for="item in displaySuggestions" :key="item.id">
              <div class="row">
                <strong>{{ item.title }}</strong>
                <span :class="['risk-pill', item.risk_level]">{{ item.risk_level }}</span>
              </div>
              <p>{{ item.reason_summary }}</p>
              <p v-if="settings.show_impact_estimate && hasImpactEstimate(item)" class="meta">
                影响预估: {{ summarizeImpact(item) }}
              </p>
              <div class="row actions-inline">
                <button class="small" :disabled="loading" @click="approveSuggestion(item.id)">Approve</button>
                <button class="small secondary" :disabled="loading" @click="rejectSuggestion(item.id)">
                  Reject
                </button>
              </div>
            </li>
          </ul>
        </div>

        <div v-else-if="activeTab === 'tasks'" class="tab-pane">
          <div class="row">
            <h3>Recent Tasks</h3>
            <button class="small" :disabled="loading" @click="refreshAll">Refresh</button>
          </div>
          <ul class="popup-list">
            <li v-for="task in tasks" :key="task.id">
              <div class="row">
                <strong>{{ task.task_type }}</strong>
                <span :class="['risk-pill', task.risk_level]">{{ task.risk_level }}</span>
              </div>
              <p>Status: {{ task.status }} · Retry: {{ task.retry_count }}</p>
              <p v-if="task.failure_reason" class="error">{{ task.failure_reason }}</p>
              <div class="row actions-inline">
                <button
                  class="small secondary"
                  :disabled="loading || task.status !== 'queued'"
                  @click="cancelTask(task.id)"
                >
                  Cancel
                </button>
                <button
                  class="small"
                  :disabled="loading || task.status !== 'failed'"
                  @click="retryTask(task.id)"
                >
                  Retry
                </button>
              </div>
            </li>
          </ul>
        </div>

        <div v-else class="tab-pane">
          <div class="row">
            <h3>Notifications</h3>
            <button class="small" :disabled="loading" @click="refreshAll">Refresh</button>
          </div>
          <ul class="popup-list">
            <li v-for="item in notifications" :key="item.id" :class="{ unread: !item.is_read }">
              <div class="row">
                <strong>{{ item.title }}</strong>
                <span :class="['risk-pill', item.priority]">{{ item.priority }}</span>
              </div>
              <p>{{ item.body }}</p>
              <div class="row actions-inline">
                <small>{{ formatDate(item.created_at) }}</small>
                <button
                  v-if="!item.is_read"
                  class="small"
                  :disabled="loading"
                  @click="markRead(item.id)"
                >
                  Mark Read
                </button>
              </div>
            </li>
          </ul>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { AdGoal, MeResponse, Notification, Suggestion, Task } from "@trademate/shared-types";
import { extensionApi } from "../shared/api";
import { DEFAULT_SETTINGS, loadExtensionSettings, type ExtensionSettings } from "../shared/settings";

const account = ref("demo@trademate.dev");
const password = ref("demo123");
const error = ref("");
const message = ref("");
const token = ref("");
const loading = ref(false);
const activeTab = ref<"overview" | "suggestions" | "tasks" | "notifications">("overview");

const me = ref<MeResponse | null>(null);
const goal = ref<AdGoal | null>(null);
const suggestions = ref<Suggestion[]>([]);
const tasks = ref<Task[]>([]);
const notifications = ref<Notification[]>([]);
const settings = ref<ExtensionSettings>({ ...DEFAULT_SETTINGS });

const displaySuggestions = computed(() => {
  if (settings.value.show_high_risk_only) {
    return suggestions.value.filter((item) => item.risk_level === "high");
  }
  return suggestions.value;
});
const topSuggestions = computed(() => displaySuggestions.value.slice(0, 5));
const highRiskCount = computed(
  () => suggestions.value.filter((item) => item.risk_level === "high").length
);
const unreadNotifications = computed(
  () => notifications.value.filter((item) => !item.is_read).length
);
const storeName = computed(() => {
  if (settings.value.default_store_id) {
    const matched = me.value?.stores.find((store) => store.id === settings.value.default_store_id);
    if (matched) {
      return matched.store_name;
    }
  }

  return me.value?.stores[0]?.store_name ?? "No store";
});
const goalSummary = computed(() => {
  if (!goal.value) {
    return "No goal loaded";
  }

  return `${goal.value.goal_name} · ACOS ${goal.value.acos_target ?? "N/A"} · ${goal.value.risk_profile}`;
});

onMounted(async () => {
  settings.value = await loadExtensionSettings();

  const storedToken = await chrome.storage.local.get("token");
  if (storedToken.token) {
    token.value = storedToken.token;
    await hydrate();
  }
});

async function submit() {
  error.value = "";
  message.value = "";
  try {
    const result = await extensionApi.login(account.value, password.value);
    token.value = result.token;
    await chrome.storage.local.set({ token: result.token });
    await hydrate();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Login failed";
  }
}

async function hydrate() {
  if (!token.value) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    const [meResult, goalResult, suggestionsResult, tasksResult, notificationsResult] = await Promise.all([
      extensionApi.me(token.value),
      extensionApi.goal(token.value),
      extensionApi.suggestions(token.value, { status: "ready", page_size: 50 }),
      extensionApi.tasks(token.value, { page_size: 30 }),
      extensionApi.notifications(token.value, { limit: 30 })
    ]);

    me.value = meResult;
    goal.value = goalResult;
    suggestions.value = suggestionsResult.list;
    tasks.value = tasksResult.list;
    notifications.value = notificationsResult.list;
    settings.value = await loadExtensionSettings();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Load failed";
  } finally {
    loading.value = false;
  }
}

async function refreshAll() {
  await hydrate();
}

async function approveSuggestion(suggestionID: string) {
  if (!token.value) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    await extensionApi.approveSuggestion(token.value, suggestionID);
    message.value = "Suggestion approved";
    await hydrate();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Approve failed";
  } finally {
    loading.value = false;
  }
}

async function rejectSuggestion(suggestionID: string) {
  if (!token.value) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    await extensionApi.rejectSuggestion(token.value, suggestionID);
    message.value = "Suggestion rejected";
    await hydrate();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Reject failed";
  } finally {
    loading.value = false;
  }
}

async function cancelTask(taskID: string) {
  if (!token.value) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    await extensionApi.cancelTask(token.value, taskID);
    message.value = "Task cancelled";
    await hydrate();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Cancel failed";
  } finally {
    loading.value = false;
  }
}

async function retryTask(taskID: string) {
  if (!token.value) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    await extensionApi.retryTask(token.value, taskID);
    message.value = "Task retry queued";
    await hydrate();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Retry failed";
  } finally {
    loading.value = false;
  }
}

async function markRead(notificationID: string) {
  if (!token.value) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    await extensionApi.markNotificationRead(token.value, notificationID);
    message.value = "Notification marked read";
    notifications.value = notifications.value.map((item) =>
      item.id === notificationID ? { ...item, is_read: true } : item
    );
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Mark read failed";
  } finally {
    loading.value = false;
  }
}

function hasImpactEstimate(item: Suggestion) {
  return item.impact_estimate_json && Object.keys(item.impact_estimate_json).length > 0;
}

function summarizeImpact(item: Suggestion) {
  if (!item.impact_estimate_json) {
    return "-";
  }

  return Object.entries(item.impact_estimate_json)
    .slice(0, 2)
    .map(([key, value]) => `${key}: ${String(value)}`)
    .join(" · ");
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}
</script>
