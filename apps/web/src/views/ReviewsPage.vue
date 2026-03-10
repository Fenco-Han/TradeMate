<template>
  <AppShell title="Review Center" :store-name="storeName">
    <section class="panel list-controls">
      <div class="filter-row split">
        <label>
          Task Status
          <select v-model="taskStatusFilter" @change="loadTasks">
            <option value="">All</option>
            <option value="succeeded">Succeeded</option>
            <option value="failed">Failed</option>
            <option value="cancelled">Cancelled</option>
            <option value="running">Running</option>
            <option value="queued">Queued</option>
          </select>
        </label>
        <label>
          Review Status
          <select v-model="reviewStatusFilter">
            <option value="">All</option>
            <option value="ready">Ready</option>
            <option value="partial">Partial</option>
            <option value="pending">Pending</option>
          </select>
        </label>
        <div class="actions-row">
          <button :disabled="loading" @click="loadTasks">Refresh</button>
        </div>
      </div>
      <div class="stats">
        <article class="stat">
          <span>Loaded Tasks</span>
          <strong>{{ totalTasks }}</strong>
        </article>
        <article class="stat">
          <span>Review Ready</span>
          <strong>{{ readyCount }}</strong>
        </article>
        <article class="stat">
          <span>Review Partial</span>
          <strong>{{ partialCount }}</strong>
        </article>
        <article class="stat">
          <span>Review Pending</span>
          <strong>{{ pendingCount }}</strong>
        </article>
      </div>
      <p v-if="message" class="hint">{{ message }}</p>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <section class="panel">
      <div v-if="loading" class="hint">Loading reviews...</div>
      <div v-else-if="filteredTasks.length === 0" class="hint">当前暂无可复盘任务</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>Task</th>
            <th>Task Status</th>
            <th>Review Status</th>
            <th>Execution Channel</th>
            <th>Finished</th>
            <th>Summary</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in filteredTasks" :key="task.id">
            <td>
              <strong>{{ task.task_type }}</strong>
              <p class="table-sub">{{ task.target_type }} / {{ task.target_id }}</p>
            </td>
            <td>{{ task.status }}</td>
            <td>{{ reviewStatusOf(task.id) }}</td>
            <td>{{ executionChannelOf(task.id) }}</td>
            <td>{{ formatDate(task.finished_at) }}</td>
            <td>{{ reviewMap[task.id]?.summary ?? "-" }}</td>
            <td class="table-actions">
              <button class="small" :disabled="loading" @click="selectTask(task.id)">View</button>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="panel" v-if="selectedReview">
      <div class="filter-row split">
        <strong>Review Detail · {{ selectedTaskID }}</strong>
        <button class="small" :disabled="loading" @click="refreshSelectedReview">Refresh Review</button>
      </div>
      <p>
        状态: <strong>{{ selectedReview.status }}</strong>
        <span v-if="selectedReview.generated_at"> · 生成时间: {{ formatDate(selectedReview.generated_at) }}</span>
      </p>
      <p v-if="selectedReview.summary">{{ selectedReview.summary }}</p>
      <div class="panel-grid metrics-grid">
        <article class="panel nested-panel">
          <h4>Before Metrics</h4>
          <pre class="json-block">{{ formatJSON(selectedReview.before_metrics) }}</pre>
        </article>
        <article class="panel nested-panel">
          <h4>After Metrics</h4>
          <pre class="json-block">{{ formatJSON(selectedReview.after_metrics) }}</pre>
        </article>
      </div>
    </section>
  </AppShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { ReviewSnapshot, Task, TaskStatus } from "@trademate/shared-types";
import AppShell from "../components/AppShell.vue";
import { api } from "../lib/api";
import { hydrateSession, sessionState } from "../stores/session";

const router = useRouter();
const route = useRoute();

const loading = ref(false);
const error = ref("");
const message = ref("");
const tasks = ref<Task[]>([]);
const reviewMap = ref<Record<string, ReviewSnapshot>>({});
const selectedTaskID = ref("");
const statusCounts = ref<Record<string, number>>({
  ready: 0,
  partial: 0,
  pending: 0
});

const taskStatusFilter = ref<"" | TaskStatus>("");
const reviewStatusFilter = ref<"" | "pending" | "partial" | "ready">("");

const storeName = computed(() => sessionState.me?.stores[0]?.store_name ?? "");

const filteredTasks = computed(() => {
  return tasks.value.filter((task) => {
    const reviewStatus = reviewStatusOf(task.id);
    if (reviewStatusFilter.value && reviewStatus !== reviewStatusFilter.value) {
      return false;
    }
    return true;
  });
});

const selectedReview = computed(() => {
  if (!selectedTaskID.value) {
    return null;
  }
  return reviewMap.value[selectedTaskID.value] ?? null;
});

const totalTasks = computed(() => tasks.value.length);
const readyCount = computed(() => statusCounts.value.ready ?? 0);
const partialCount = computed(() => statusCounts.value.partial ?? 0);
const pendingCount = computed(() => Math.max(0, totalTasks.value - readyCount.value - partialCount.value));

onMounted(async () => {
  if (!sessionState.token) {
    await router.push("/login");
    return;
  }

  if (!sessionState.me) {
    await hydrateSession();
  }

  const queryTaskID = typeof route.query.task_id === "string" ? route.query.task_id : "";
  if (queryTaskID) {
    selectedTaskID.value = queryTaskID;
  }

  await loadTasks();
});

async function loadTasks() {
  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    const [taskResult, reviewResult] = await Promise.all([
      api.listTasks({
        status: taskStatusFilter.value || undefined,
        page: 1,
        page_size: 100
      }),
      api.listTaskReviews({
        limit: 200
      })
    ]);
    tasks.value = taskResult.list;

    const nextMap: Record<string, ReviewSnapshot> = {};
    for (const item of reviewResult.list) {
      nextMap[item.task_id] = item;
    }
    reviewMap.value = nextMap;
    statusCounts.value = reviewResult.status_counts ?? { ready: 0, partial: 0, pending: 0 };

    if (!selectedTaskID.value && taskResult.list.length > 0) {
      selectTask(taskResult.list[0].id);
    } else if (selectedTaskID.value && !taskResult.list.some((item) => item.id === selectedTaskID.value)) {
      selectedTaskID.value = "";
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load reviews";
  } finally {
    loading.value = false;
  }
}

function selectTask(taskID: string) {
  selectedTaskID.value = taskID;
  void router.replace({ query: { ...route.query, task_id: taskID } });
}

async function refreshSelectedReview() {
  if (!selectedTaskID.value) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    await loadTasks();
    message.value = "Review refreshed";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to refresh review";
  } finally {
    loading.value = false;
  }
}

function reviewStatusOf(taskID: string) {
  return reviewMap.value[taskID]?.status ?? "pending";
}

function executionChannelOf(taskID: string) {
  const review = reviewMap.value[taskID];
  if (!review) {
    return "-";
  }

  const afterChannel = metricString(review.after_metrics, "execution_channel");
  if (afterChannel) {
    return afterChannel;
  }

  if (metricBoolean(review.after_metrics, "fallback_requested") || metricBoolean(review.before_metrics, "fallback_requested")) {
    return "browser_fallback(planned)";
  }

  return "api";
}

function metricString(metrics: Record<string, unknown> | undefined, key: string) {
  const value = metrics?.[key];
  return typeof value === "string" ? value : "";
}

function metricBoolean(metrics: Record<string, unknown> | undefined, key: string) {
  return metrics?.[key] === true;
}

function formatDate(value?: string | null) {
  if (!value) {
    return "-";
  }
  return new Date(value).toLocaleString();
}

function formatJSON(value?: Record<string, unknown>) {
  return JSON.stringify(value ?? {}, null, 2);
}
</script>
