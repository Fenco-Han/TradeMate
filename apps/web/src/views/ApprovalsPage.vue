<template>
  <AppShell title="Pending Approvals" :store-name="storeName">
    <section class="panel list-controls">
      <div class="filter-row">
        <label>
          Risk Level
          <select v-model="riskLevel" @change="loadSuggestions">
            <option value="">All</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
          </select>
        </label>
        <button :disabled="loading" @click="loadSuggestions">Refresh</button>
      </div>
      <div class="actions-row">
        <button :disabled="selectedIDs.length === 0 || loading" @click="batchApprove">
          Batch Approve ({{ selectedIDs.length }})
        </button>
        <button class="secondary" :disabled="selectedIDs.length === 0 || loading" @click="batchReject">
          Batch Reject ({{ selectedIDs.length }})
        </button>
        <span v-if="message" class="hint">{{ message }}</span>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <section class="panel">
      <div v-if="loading" class="hint">Loading suggestions...</div>
      <div v-else-if="suggestions.length === 0" class="hint">当前无待处理建议</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>
              <input type="checkbox" :checked="allSelected" @change="toggleSelectAll" />
            </th>
            <th>Title</th>
            <th>Type</th>
            <th>Risk</th>
            <th>Status</th>
            <th>Created</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in suggestions" :key="item.id">
            <td>
              <input
                type="checkbox"
                :checked="selectedIDs.includes(item.id)"
                @change="toggleItem(item.id)"
              />
            </td>
            <td>
              <strong>{{ item.title }}</strong>
              <p class="table-sub">{{ item.reason_summary }}</p>
            </td>
            <td>{{ item.suggestion_type }}</td>
            <td><span :class="['risk-tag', item.risk_level]">{{ item.risk_level }}</span></td>
            <td>{{ item.status }}</td>
            <td>{{ formatDate(item.created_at) }}</td>
            <td class="table-actions">
              <button class="small" :disabled="loading" @click="approveOne(item.id)">Approve</button>
              <button class="small secondary" :disabled="loading" @click="rejectOne(item.id)">Reject</button>
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
import type { RiskLevel, Suggestion } from "@trademate/shared-types";
import AppShell from "../components/AppShell.vue";
import { api } from "../lib/api";
import { hydrateSession, sessionState } from "../stores/session";

const router = useRouter();

const loading = ref(false);
const error = ref("");
const message = ref("");
const suggestions = ref<Suggestion[]>([]);
const selectedIDs = ref<string[]>([]);
const riskLevel = ref<"" | RiskLevel>("");

const storeName = computed(() => sessionState.me?.stores[0]?.store_name ?? "");
const allSelected = computed(
  () => suggestions.value.length > 0 && selectedIDs.value.length === suggestions.value.length
);

onMounted(async () => {
  if (!sessionState.token) {
    await router.push("/login");
    return;
  }

  if (!sessionState.me) {
    await hydrateSession();
  }

  await loadSuggestions();
});

async function loadSuggestions() {
  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    const result = await api.listSuggestions({
      status: "ready",
      risk_level: riskLevel.value || undefined,
      page: 1,
      page_size: 100
    });
    suggestions.value = result.list;
    selectedIDs.value = selectedIDs.value.filter((id) => result.list.some((item) => item.id === id));
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load suggestions";
  } finally {
    loading.value = false;
  }
}

async function approveOne(suggestionID: string) {
  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    await api.approveSuggestion(suggestionID, { execute_immediately: true });
    message.value = "Suggestion approved";
    await loadSuggestions();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Approve failed";
  } finally {
    loading.value = false;
  }
}

async function rejectOne(suggestionID: string) {
  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    await api.rejectSuggestion(suggestionID);
    message.value = "Suggestion rejected";
    await loadSuggestions();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Reject failed";
  } finally {
    loading.value = false;
  }
}

async function batchApprove() {
  if (selectedIDs.value.length === 0) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    const result = await api.batchApproveSuggestions({
      suggestion_ids: selectedIDs.value,
      execute_immediately: true
    });
    message.value = `Approved ${result.total} suggestion(s)`;
    selectedIDs.value = [];
    await loadSuggestions();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Batch approve failed";
  } finally {
    loading.value = false;
  }
}

async function batchReject() {
  if (selectedIDs.value.length === 0) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";
  try {
    const result = await api.batchRejectSuggestions({
      suggestion_ids: selectedIDs.value
    });
    message.value = `Rejected ${result.total} suggestion(s)`;
    selectedIDs.value = [];
    await loadSuggestions();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Batch reject failed";
  } finally {
    loading.value = false;
  }
}

function toggleItem(suggestionID: string) {
  if (selectedIDs.value.includes(suggestionID)) {
    selectedIDs.value = selectedIDs.value.filter((id) => id !== suggestionID);
    return;
  }

  selectedIDs.value = [...selectedIDs.value, suggestionID];
}

function toggleSelectAll() {
  if (allSelected.value) {
    selectedIDs.value = [];
    return;
  }

  selectedIDs.value = suggestions.value.map((item) => item.id);
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}
</script>
