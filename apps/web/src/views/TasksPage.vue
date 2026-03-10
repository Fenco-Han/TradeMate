<template>
  <AppShell title="Tasks Center" :store-name="storeName">
    <section class="panel list-controls">
      <div class="filter-row split">
        <label>
          Status
          <select v-model="statusFilter" @change="loadTasks">
            <option value="">All</option>
            <option value="queued">Queued</option>
            <option value="running">Running</option>
            <option value="succeeded">Succeeded</option>
            <option value="failed">Failed</option>
            <option value="cancelled">Cancelled</option>
          </select>
        </label>
        <label>
          Risk Level
          <select v-model="riskLevel" @change="loadTasks">
            <option value="">All</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
          </select>
        </label>
        <button :disabled="loading" @click="loadTasks">Refresh</button>
        <button :disabled="loading" @click="runWorkerOnce">Run Once</button>
      </div>
      <p v-if="message" class="hint">{{ message }}</p>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <section class="panel">
      <div v-if="loading" class="hint">Loading tasks...</div>
      <div v-else-if="tasks.length === 0" class="hint">暂未生成执行任务</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>Task</th>
            <th>Risk</th>
            <th>Status</th>
            <th>Retry</th>
            <th>Created</th>
            <th>Failure Reason</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in tasks" :key="task.id">
            <td>
              <strong>{{ task.task_type }}</strong>
              <p class="table-sub">{{ task.target_type }} / {{ task.target_id }}</p>
            </td>
            <td><span :class="['risk-tag', task.risk_level]">{{ task.risk_level }}</span></td>
            <td>{{ task.status }}</td>
            <td>{{ task.retry_count }}</td>
            <td>{{ formatDate(task.created_at) }}</td>
            <td>{{ task.failure_reason || "-" }}</td>
            <td class="table-actions">
              <button class="small" :disabled="loading" @click="openReview(task.id)">Review</button>
              <button class="small secondary" :disabled="loading" @click="openDetail(task.id)">Detail</button>
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
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <section class="panel" v-if="selectedDetail">
      <div class="filter-row split">
        <strong>Task Detail · {{ selectedDetail.task.id }}</strong>
        <button class="small" :disabled="loading || detailLoading" @click="refreshDetail">Refresh Detail</button>
      </div>
      <p class="hint" v-if="detailLoading">Loading task detail...</p>
      <div class="panel-grid">
        <article class="panel nested-panel">
          <h4>Execution</h4>
          <p>Review Status: <strong>{{ selectedDetail.review_status }}</strong></p>
          <p>Channel: <strong>{{ selectedDetail.execution.channel }}</strong></p>
          <p>Status: <strong>{{ selectedDetail.execution.status }}</strong></p>
          <p>Fallback Requested: <strong>{{ selectedDetail.execution.fallback_requested ? "yes" : "no" }}</strong></p>
          <p>Fallback Used: <strong>{{ selectedDetail.execution.fallback_used ? "yes" : "no" }}</strong></p>
          <p>Execution ID: <strong>{{ selectedDetail.execution.execution_id || "-" }}</strong></p>
          <p>Executed At: <strong>{{ formatDate(selectedDetail.task.executed_at) }}</strong></p>
          <p>Finished At: <strong>{{ formatDate(selectedDetail.task.finished_at) }}</strong></p>
        </article>
        <article class="panel nested-panel">
          <h4>Recent Events</h4>
          <div v-if="selectedDetail.task_events.length === 0" class="hint">No events</div>
          <ul v-else class="inline-list">
            <li v-for="event in selectedDetail.task_events.slice(-8).reverse()" :key="event.id">
              <strong>{{ event.to_status }}</strong>
              <span class="table-sub"> · {{ event.event_type }} · {{ formatDate(event.created_at) }}</span>
              <p class="table-sub">{{ summarizeEvent(event.event_payload_json) }}</p>
            </li>
          </ul>
        </article>
      </div>
      <article class="panel nested-panel detail-audit-panel">
        <h4>Related Audit Logs</h4>
        <div v-if="relatedAuditLogs.length === 0" class="hint">No audit logs bound to this task</div>
        <ul v-else class="inline-list">
          <li v-for="item in relatedAuditLogs" :key="item.id">
            <strong>{{ item.action }}</strong>
            <span class="table-sub"> · {{ item.result }} · {{ formatDate(item.created_at) }}</span>
            <p class="table-sub">{{ item.metadata_json || "-" }}</p>
          </li>
        </ul>
      </article>

      <article class="panel nested-panel detail-audit-panel" v-if="selectedReview">
        <h4>Review Snapshot</h4>
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
      </article>
    </section>
  </AppShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { ReviewSnapshot, RiskLevel, Task, TaskDetailResponse, TaskStatus } from "@trademate/shared-types";
import AppShell from "../components/AppShell.vue";
import { api } from "../lib/api";
import { hydrateSession, sessionState } from "../stores/session";

const router = useRouter();
const route = useRoute();

const loading = ref(false);
const detailLoading = ref(false);
const error = ref("");
const message = ref("");
const tasks = ref<Task[]>([]);
const statusFilter = ref<"" | TaskStatus>("");
const riskLevel = ref<"" | RiskLevel>("");
const selectedTaskID = ref("");
const selectedDetail = ref<TaskDetailResponse | null>(null);
const selectedReview = ref<ReviewSnapshot | null>(null);

const storeName = computed(() => sessionState.me?.stores[0]?.store_name ?? "");
const relatedAuditLogs = computed(() => {
  if (!selectedDetail.value) {
    return [];
  }
  return selectedDetail.value.audit_logs
    .filter((item) => item.target_type === "task" && item.target_id === selectedDetail.value?.task.id)
    .slice(0, 10);
});

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

  if (selectedTaskID.value) {
    await loadTaskDetail(selectedTaskID.value);
  }
});

async function loadTasks() {
  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    const result = await api.listTasks({
      status: statusFilter.value || undefined,
      risk_level: riskLevel.value || undefined,
      page: 1,
      page_size: 100
    });
    tasks.value = result.list;
    if (selectedTaskID.value && !tasks.value.some((item) => item.id === selectedTaskID.value)) {
      selectedTaskID.value = "";
      selectedDetail.value = null;
      selectedReview.value = null;
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load tasks";
  } finally {
    loading.value = false;
  }
}

async function cancelTask(taskID: string) {
  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    await api.cancelTask(taskID);
    message.value = "Task cancelled";
    await loadTasks();
    if (selectedTaskID.value) {
      await loadTaskDetail(selectedTaskID.value);
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Cancel failed";
  } finally {
    loading.value = false;
  }
}

async function retryTask(taskID: string) {
  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    await api.retryTask(taskID);
    message.value = "Task queued for retry";
    await loadTasks();
    if (selectedTaskID.value) {
      await loadTaskDetail(selectedTaskID.value);
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Retry failed";
  } finally {
    loading.value = false;
  }
}

async function runWorkerOnce() {
  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    const result = await api.runTasksOnce({ limit: 20 });
    message.value = `Worker执行完成：成功 ${result.succeeded}，失败 ${result.failed}，跳过 ${result.skipped}`;
    await loadTasks();
    if (selectedTaskID.value) {
      await loadTaskDetail(selectedTaskID.value);
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Run worker failed";
  } finally {
    loading.value = false;
  }
}

function formatDate(value?: string | null) {
  if (!value) {
    return "-";
  }
  return new Date(value).toLocaleString();
}

function openReview(taskID: string) {
  void router.push({ path: "/reviews", query: { task_id: taskID } });
}

async function openDetail(taskID: string) {
  selectedTaskID.value = taskID;
  await router.replace({ query: { ...route.query, task_id: taskID } });
  await loadTaskDetail(taskID);
}

async function refreshDetail() {
  if (!selectedTaskID.value) {
    return;
  }
  await loadTaskDetail(selectedTaskID.value);
}

async function loadTaskDetail(taskID: string) {
  detailLoading.value = true;
  error.value = "";
  try {
    const [detail, review] = await Promise.all([api.getTask(taskID), api.getTaskReview(taskID)]);
    selectedDetail.value = detail;
    selectedReview.value = review;
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load task detail";
  } finally {
    detailLoading.value = false;
  }
}

function summarizeEvent(raw?: string | null) {
  if (!raw) {
    return "-";
  }
  try {
    const payload = JSON.parse(raw) as Record<string, unknown>;
    if (typeof payload.reason === "string" && payload.reason.trim() !== "") {
      return payload.reason;
    }
  } catch {
    return raw;
  }
  return raw;
}

function formatJSON(value?: Record<string, unknown>) {
  return JSON.stringify(value ?? {}, null, 2);
}
</script>
