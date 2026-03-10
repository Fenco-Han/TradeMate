<template>
  <AppShell title="Ad Agent Dashboard" :store-name="storeName">
    <div class="panel-grid">
      <section class="panel">
        <h2>Today</h2>
        <div class="stats">
          <div class="stat">
            <span>Suggestions</span>
            <strong>{{ suggestions.length }}</strong>
          </div>
          <div class="stat">
            <span>High Risk</span>
            <strong>{{ highRiskCount }}</strong>
          </div>
        </div>
      </section>
      <section class="panel">
        <h2>Current Goal</h2>
        <p>{{ goalSummary }}</p>
      </section>
    </div>

    <section class="panel">
      <h2>Suggestions</h2>
      <ul class="suggestions">
        <li v-for="suggestion in suggestions" :key="suggestion.id" class="suggestion-card">
          <div class="suggestion-top">
            <strong>{{ suggestion.title }}</strong>
            <span :class="['risk-tag', suggestion.risk_level]">{{ suggestion.risk_level }}</span>
          </div>
          <p>{{ suggestion.reason_summary }}</p>
        </li>
      </ul>
    </section>
  </AppShell>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import AppShell from "../components/AppShell.vue";
import { hydrateSession, sessionState } from "../stores/session";

const router = useRouter();

const suggestions = computed(() => sessionState.suggestions);
const highRiskCount = computed(() =>
  sessionState.suggestions.filter((item) => item.risk_level === "high").length
);
const goalSummary = computed(() => {
  if (!sessionState.goal) {
    return "No goal loaded";
  }

  return `${sessionState.goal.goal_name} · ACOS ${sessionState.goal.acos_target ?? "N/A"} · ${sessionState.goal.risk_profile}`;
});
const storeName = computed(() => sessionState.me?.stores[0]?.store_name ?? "");

onMounted(async () => {
  if (!sessionState.token) {
    await router.push("/login");
    return;
  }

  if (!sessionState.me) {
    await hydrateSession();
  }
});
</script>
