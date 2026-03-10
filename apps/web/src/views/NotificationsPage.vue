<template>
  <AppShell title="Notifications" :store-name="storeName">
    <section class="panel list-controls">
      <div class="filter-row split">
        <label>
          Read Status
          <select v-model="readFilter">
            <option value="all">All</option>
            <option value="unread">Unread</option>
            <option value="read">Read</option>
          </select>
        </label>
        <div class="actions-row">
          <button :disabled="loading" @click="loadNotifications">Refresh</button>
          <button :disabled="loading || unreadCount === 0" @click="markAllVisibleRead">
            Mark Visible Read
          </button>
        </div>
      </div>
      <p v-if="message" class="hint">{{ message }}</p>
      <p v-if="error" class="error">{{ error }}</p>
    </section>

    <section class="panel">
      <div v-if="loading" class="hint">Loading notifications...</div>
      <div v-else-if="filteredNotifications.length === 0" class="hint">当前暂无通知</div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th>Title</th>
            <th>Priority</th>
            <th>Status</th>
            <th>Created</th>
            <th>Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in filteredNotifications" :key="item.id">
            <td>
              <strong>{{ item.title }}</strong>
              <p class="table-sub">{{ item.body }}</p>
            </td>
            <td><span :class="['risk-tag', item.priority]">{{ item.priority }}</span></td>
            <td>{{ item.is_read ? "read" : "unread" }}</td>
            <td>{{ formatDate(item.created_at) }}</td>
            <td class="table-actions">
              <button
                class="small"
                :disabled="loading || item.is_read"
                @click="markRead(item.id)"
              >
                Mark Read
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
import type { Notification } from "@trademate/shared-types";
import AppShell from "../components/AppShell.vue";
import { api } from "../lib/api";
import { hydrateSession, sessionState } from "../stores/session";

const router = useRouter();

const loading = ref(false);
const error = ref("");
const message = ref("");
const notifications = ref<Notification[]>([]);
const readFilter = ref<"all" | "read" | "unread">("all");

const storeName = computed(() => sessionState.me?.stores[0]?.store_name ?? "");

const filteredNotifications = computed(() => {
  if (readFilter.value === "read") {
    return notifications.value.filter((item) => item.is_read);
  }
  if (readFilter.value === "unread") {
    return notifications.value.filter((item) => !item.is_read);
  }
  return notifications.value;
});

const unreadCount = computed(
  () => filteredNotifications.value.filter((item) => !item.is_read).length
);

onMounted(async () => {
  if (!sessionState.token) {
    await router.push("/login");
    return;
  }

  if (!sessionState.me) {
    await hydrateSession();
  }

  await loadNotifications();
});

async function loadNotifications() {
  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    const result = await api.listNotifications({ limit: 100 });
    notifications.value = result.list;
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load notifications";
  } finally {
    loading.value = false;
  }
}

async function markRead(notificationID: string) {
  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    await api.markNotificationRead(notificationID);
    notifications.value = notifications.value.map((item) =>
      item.id === notificationID ? { ...item, is_read: true } : item
    );
    message.value = "Notification marked as read";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Mark read failed";
  } finally {
    loading.value = false;
  }
}

async function markAllVisibleRead() {
  const targets = filteredNotifications.value.filter((item) => !item.is_read);
  if (targets.length === 0) {
    return;
  }

  loading.value = true;
  error.value = "";
  message.value = "";

  try {
    await Promise.all(targets.map((item) => api.markNotificationRead(item.id)));
    const targetIDs = new Set(targets.map((item) => item.id));
    notifications.value = notifications.value.map((item) =>
      targetIDs.has(item.id) ? { ...item, is_read: true } : item
    );
    message.value = `Marked ${targets.length} notification(s) read`;
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Batch mark read failed";
  } finally {
    loading.value = false;
  }
}

function formatDate(value: string) {
  return new Date(value).toLocaleString();
}
</script>
