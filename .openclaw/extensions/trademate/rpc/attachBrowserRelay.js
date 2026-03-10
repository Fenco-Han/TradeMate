import { buildAttachResult } from "../background/relay_state.js";

export function handleAttachBrowserRelay(params = {}) {
  return buildAttachResult(params);
}
