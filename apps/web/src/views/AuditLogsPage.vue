<template>
  <AppShell title="Audit Logs" :store-name="storeName">
    <section class="panel list-controls">
      <div class="filter-row split">
        <label>
          Search
          <input v-model="keyword" placeholder="action / target / actor" />
        </label>
        <div class="actions-row">
          <button :disabled="loading" @click="loadAuditLogs">Refresh</button>
        </div>
      </div>
      <p v-if="message" class="hint">{{ message }}</p>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <section class="panel">
      <div v-if="loading" class="hint">Loading audit logs...</div>
      <div v-else-if="filteredLogs.length === 0" class="hint">当前暂无审计记录</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>Action</th>
            <th>Actor</th>
            <th>Target</th>
            <th>Result</th>
            <th>Created</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in filteredLogs" :key="item.id">
            <td>
              <strong>{{ item.action }}</strong>
              <p v-if="item.metadata_json" class="table-sub">{{ item.metadata_json }}</p>
            </td>
            <td>{{ item.actor_id }}</td>
            <td>{{ item.target_type }} / {{ item.target_id }}</td>
            <td>{{ item.result }}</td>
            <td>{{ formatDate(item.created_at) }}</td>
          </tr>
        </tbody>
      </table>
    </section>
  </AppShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import type { AuditLog } from "@trademate/shared-types";
import AppShell from "../components/AppShell.vue";
import { api } from "../lib/api";
import { hydrateSession, sessionState } from "../stores/session";

const router = useRouter();

const loading = ref(false);
const error = ref("");
const message = ref("");
const keyword = ref("");
const logs = ref<AuditLog[]>([]);

const storeName = computed(() => sessionState.me?.stores[0]?.store_name ?? "");

const filteredLogs = computed(() => {
  const normalizedKeyword = keyword.value.trim().toLowerCase();
  if (normalizedKeyword === "") {
    return logs.value;
  }

  return logs.value.filter((item) => {
    return (
      item.action.toLowerCase().includes(normalizedKeyword) ||
      item.actor_id.toLowerCase().includes(normalizedKeyword) ||
      item.target_type.toLowerCase().includes(normalizedKeyword) ||
      item.target_id.toLowerCase().includes(normalizedKeyword)
    );
  });
});

onMounted(async () => {
  if (!sessionState.token) {
    await router.push("/login");
    return;
  }

  if (!sessionState.me) {
    await hydrateSession();
  }

  await loadAuditLogs();
});

async function loadAuditLogs() {
  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    const result = await api.listAuditLogs({ limit: 200 });
    logs.value = result.list;
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load audit logs";
  } finally {
    loading.value = false;
  }
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}
</script>
