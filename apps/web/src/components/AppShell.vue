<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">TradeMate</div>
      <router-link to="/dashboard">Dashboard</router-link>
      <router-link to="/approvals">Approvals</router-link>
      <router-link to="/tasks">Tasks</router-link>
      <router-link to="/goals">Goals</router-link>
    </aside>
    <main class="content">
      <header class="topbar">
        <div>
          <strong>{{ title }}</strong>
        </div>
        <div v-if="storeName" class="store-chip">{{ storeName }}</div>
      </header>
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  storeName?: string;
}>();
</script>
