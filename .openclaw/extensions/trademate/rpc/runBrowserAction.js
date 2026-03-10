import { runBrowserAction } from "../browser/actions.js";
import { evaluateRelayStatus } from "../background/relay_state.js";

export async function handleRunBrowserAction(params = {}) {
  const relayStatus = evaluateRelayStatus({
    attached: params.attached,
    tab_id: params.tab_id,
    url: params.url,
    browser: params.browser
  });

  if (!relayStatus.attached) {
    return {
      accepted: false,
      error_code: relayStatus.error_code,
      message: "relay is not attached"
    };
  }

  try {
    const result = await runBrowserAction(params.action_name, relayStatus, params.payload || {});
    return {
      ...result,
      accepted: true
    };
  } catch (err) {
    return {
      accepted: false,
      error_code: err?.code || "ACTION_FAILED",
      message: err?.message || "browser action failed"
    };
  }
}
