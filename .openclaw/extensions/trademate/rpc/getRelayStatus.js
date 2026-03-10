import { evaluateRelayStatus } from "../background/relay_state.js";

export function handleGetRelayStatus(params = {}) {
  return evaluateRelayStatus(params);
}
