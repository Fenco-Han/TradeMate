<template>
  <div class="login-page">
    <form class="panel login-form" @submit.prevent="submit">
      <h1>TradeMate Web</h1>
      <p>Use the seeded demo account to enter the first vertical slice.</p>
      <label>
        Account
        <input v-model="account" />
      </label>
      <label>
        Password
        <input v-model="password" type="password" />
      </label>
      <button type="submit">Login</button>
      <p class="hint">demo@trademate.dev / demo123</p>
      <p v-if="error" class="error">{{ error }}</p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { login } from "../stores/session";

const router = useRouter();
const account = ref("demo@trademate.dev");
const password = ref("demo123");
const error = ref("");

async function submit() {
  error.value = "";
  try {
    await login(account.value, password.value);
    await router.push("/dashboard");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Login failed";
  }
}
</script>

