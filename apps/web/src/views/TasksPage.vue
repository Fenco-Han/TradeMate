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
  </AppShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import type { RiskLevel, Task, TaskStatus } from "@trademate/shared-types";
import AppShell from "../components/AppShell.vue";
import { api } from "../lib/api";
import { hydrateSession, sessionState } from "../stores/session";

const router = useRouter();

const loading = ref(false);
const error = ref("");
const message = ref("");
const tasks = ref<Task[]>([]);
const statusFilter = ref<"" | TaskStatus>("");
const riskLevel = ref<"" | RiskLevel>("");

const storeName = computed(() => sessionState.me?.stores[0]?.store_name ?? "");

onMounted(async () => {
  if (!sessionState.token) {
    await router.push("/login");
    return;
  }

  if (!sessionState.me) {
    await hydrateSession();
  }

  await loadTasks();
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
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Run worker failed";
  } finally {
    loading.value = false;
  }
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}
</script>
