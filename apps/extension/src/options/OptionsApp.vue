<template>
  <main class="options-shell">
    <section class="card options-card">
      <h1>TradeMate 插件设置</h1>
      <p class="meta">管理默认店铺、提醒策略、展示偏好和 Web 跳转入口。</p>

      <div class="settings-grid">
        <label>
          默认店铺 ID
          <input v-model="form.default_store_id" placeholder="store_us_001" />
        </label>

        <label>
          默认站点
          <select v-model="form.default_site_code">
            <option value="US">US</option>
            <option value="CA">CA</option>
            <option value="UK">UK</option>
            <option value="DE">DE</option>
            <option value="FR">FR</option>
            <option value="IT">IT</option>
            <option value="ES">ES</option>
            <option value="JP">JP</option>
          </select>
        </label>

        <label>
          Web 默认打开页
          <select v-model="form.default_web_target">
            <option value="dashboard">Dashboard</option>
            <option value="approvals">Approvals</option>
            <option value="tasks">Tasks</option>
            <option value="notifications">Notifications</option>
          </select>
        </label>
      </div>

      <div class="settings-grid toggles-grid">
        <label class="checkbox-row">
          <input v-model="form.reminders_enabled" type="checkbox" />
          开启通知提醒
        </label>
        <label class="checkbox-row">
          <input v-model="form.show_high_risk_only" type="checkbox" />
          仅展示高风险建议
        </label>
        <label class="checkbox-row">
          <input v-model="form.show_impact_estimate" type="checkbox" />
          展示建议影响预估
        </label>
      </div>

      <div class="actions-row">
        <button :disabled="saving" @click="save">保存设置</button>
        <button class="secondary" :disabled="saving" @click="reset">恢复默认</button>
      </div>

      <p v-if="message" class="hint">{{ message }}</p>
      <p v-if="error" class="error">{{ error }}</p>
    </section>
  </main>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import {
  DEFAULT_SETTINGS,
  loadExtensionSettings,
  resetExtensionSettings,
  saveExtensionSettings,
  type ExtensionSettings
} from "../shared/settings";

const form = reactive<ExtensionSettings>({ ...DEFAULT_SETTINGS });
const saving = ref(false);
const message = ref("");
const error = ref("");

onMounted(async () => {
  await load();
});

async function load() {
  error.value = "";
  const settings = await loadExtensionSettings();
  form.default_store_id = settings.default_store_id;
  form.default_site_code = settings.default_site_code;
  form.reminders_enabled = settings.reminders_enabled;
  form.show_high_risk_only = settings.show_high_risk_only;
  form.show_impact_estimate = settings.show_impact_estimate;
  form.default_web_target = settings.default_web_target;
}

async function save() {
  saving.value = true;
  error.value = "";
  message.value = "";

  try {
    await saveExtensionSettings({
      default_store_id: form.default_store_id.trim(),
      default_site_code: form.default_site_code,
      reminders_enabled: form.reminders_enabled,
      show_high_risk_only: form.show_high_risk_only,
      show_impact_estimate: form.show_impact_estimate,
      default_web_target: form.default_web_target
    });
    message.value = "设置已保存";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "保存失败";
  } finally {
    saving.value = false;
  }
}

async function reset() {
  saving.value = true;
  error.value = "";
  message.value = "";

  try {
    const settings = await resetExtensionSettings();
    form.default_store_id = settings.default_store_id;
    form.default_site_code = settings.default_site_code;
    form.reminders_enabled = settings.reminders_enabled;
    form.show_high_risk_only = settings.show_high_risk_only;
    form.show_impact_estimate = settings.show_impact_estimate;
    form.default_web_target = settings.default_web_target;
    message.value = "已恢复默认设置";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "恢复默认失败";
  } finally {
    saving.value = false;
  }
}
</script>
