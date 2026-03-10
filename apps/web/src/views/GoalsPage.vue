<template>
  <AppShell title="Goal Settings" :store-name="storeName">
    <form class="panel goal-form" @submit.prevent="submit">
      <label>
        Goal Name
        <input v-model="form.goal_name" />
      </label>
      <label>
        ACOS Target
        <input v-model="form.acos_target" />
      </label>
      <label>
        Daily Budget Cap
        <input v-model="form.daily_budget_cap" />
      </label>
      <label>
        Risk Profile
        <select v-model="form.risk_profile">
          <option value="conservative">Conservative</option>
          <option value="balanced">Balanced</option>
          <option value="aggressive">Aggressive</option>
        </select>
      </label>
      <label class="checkbox">
        <input v-model="form.auto_approve_enabled" type="checkbox" />
        Auto approve medium-risk changes within threshold
      </label>
      <label>
        Budget Delta Threshold
        <input v-model="form.auto_approve_budget_delta_pct" />
      </label>
      <label>
        Bid Delta Threshold
        <input v-model="form.auto_approve_bid_delta_pct" />
      </label>
      <button type="submit">Save Goal</button>
      <p v-if="message" class="hint">{{ message }}</p>
    </form>
  </AppShell>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import AppShell from "../components/AppShell.vue";
import { hydrateSession, saveGoal, sessionState } from "../stores/session";

const router = useRouter();
const message = ref("");

const form = reactive({
  goal_name: "",
  acos_target: "",
  daily_budget_cap: "",
  risk_profile: "balanced",
  auto_approve_enabled: true,
  auto_approve_budget_delta_pct: "",
  auto_approve_bid_delta_pct: ""
});

const storeName = computed(() => sessionState.me?.stores[0]?.store_name ?? "");

onMounted(async () => {
  if (!sessionState.token) {
    await router.push("/login");
    return;
  }

  if (!sessionState.me || !sessionState.goal) {
    await hydrateSession();
  }

  if (sessionState.goal) {
    form.goal_name = sessionState.goal.goal_name;
    form.acos_target = sessionState.goal.acos_target ?? "";
    form.daily_budget_cap = sessionState.goal.daily_budget_cap ?? "";
    form.risk_profile = sessionState.goal.risk_profile;
    form.auto_approve_enabled = sessionState.goal.auto_approve_enabled;
    form.auto_approve_budget_delta_pct = sessionState.goal.auto_approve_budget_delta_pct ?? "";
    form.auto_approve_bid_delta_pct = sessionState.goal.auto_approve_bid_delta_pct ?? "";
  }
});

async function submit() {
  await saveGoal({
    goal_name: form.goal_name,
    acos_target: form.acos_target || null,
    daily_budget_cap: form.daily_budget_cap || null,
    risk_profile: form.risk_profile,
    auto_approve_enabled: form.auto_approve_enabled,
    auto_approve_budget_delta_pct: form.auto_approve_budget_delta_pct || null,
    auto_approve_bid_delta_pct: form.auto_approve_bid_delta_pct || null
  });

  message.value = "Goal updated";
}
</script>
