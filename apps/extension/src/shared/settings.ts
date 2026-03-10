export interface ExtensionSettings {
  default_store_id: string;
  default_site_code: string;
  reminders_enabled: boolean;
  show_high_risk_only: boolean;
  show_impact_estimate: boolean;
  default_web_target: "dashboard" | "approvals" | "tasks" | "notifications";
}

export const SETTINGS_STORAGE_KEY = "trademate.settings";

export const DEFAULT_SETTINGS: ExtensionSettings = {
  default_store_id: "",
  default_site_code: "US",
  reminders_enabled: true,
  show_high_risk_only: false,
  show_impact_estimate: true,
  default_web_target: "dashboard"
};

export async function loadExtensionSettings(): Promise<ExtensionSettings> {
  const data = await chrome.storage.local.get(SETTINGS_STORAGE_KEY);
  const raw = data[SETTINGS_STORAGE_KEY] as Partial<ExtensionSettings> | undefined;

  return {
    ...DEFAULT_SETTINGS,
    ...(raw ?? {})
  };
}

export async function saveExtensionSettings(settings: ExtensionSettings): Promise<void> {
  await chrome.storage.local.set({
    [SETTINGS_STORAGE_KEY]: settings
  });
}

export async function resetExtensionSettings(): Promise<ExtensionSettings> {
  await saveExtensionSettings(DEFAULT_SETTINGS);
  return { ...DEFAULT_SETTINGS };
}
