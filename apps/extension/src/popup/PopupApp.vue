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
            <strong>{{ suggestions.length }}</strong>
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
      </section>

      <section class="card">
        <h2>Top suggestions</h2>
        <ul class="popup-list">
          <li v-for="item in topSuggestions" :key="item.id">
            <div class="row">
              <strong>{{ item.title }}</strong>
              <span :class="['risk-pill', item.risk_level]">{{ item.risk_level }}</span>
            </div>
            <p>{{ item.reason_summary }}</p>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import type { AdGoal, MeResponse, Suggestion } from "@trademate/shared-types";
import { extensionApi } from "../shared/api";

const account = ref("demo@trademate.dev");
const password = ref("demo123");
const error = ref("");
const token = ref("");
const me = ref<MeResponse | null>(null);
const goal = ref<AdGoal | null>(null);
const suggestions = ref<Suggestion[]>([]);

const topSuggestions = computed(() => suggestions.value.slice(0, 5));
const highRiskCount = computed(
  () => suggestions.value.filter((item) => item.risk_level === "high").length
);
const storeName = computed(() => me.value?.stores[0]?.store_name ?? "No store");
const goalSummary = computed(() => {
  if (!goal.value) {
    return "No goal loaded";
  }

  return `${goal.value.goal_name} · ACOS ${goal.value.acos_target ?? "N/A"} · ${goal.value.risk_profile}`;
});

onMounted(async () => {
  const storedToken = await chrome.storage.local.get("token");
  if (storedToken.token) {
    token.value = storedToken.token;
    await hydrate();
  }
});

async function submit() {
  error.value = "";
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

  me.value = await extensionApi.me(token.value);
  goal.value = await extensionApi.goal(token.value);
  const result = await extensionApi.suggestions(token.value);
  suggestions.value = result.list;
}
</script>
