function fail(code, message, detail = {}) {
  const err = new Error(message);
  err.code = code;
  err.detail = detail;
  return err;
}

export function validateRelayContext(ctx) {
  if (!ctx || ctx.attached !== true) {
    throw fail("RELAY_NOT_ATTACHED", "relay is not attached", { ctx });
  }
  if (!ctx.url || !ctx.url.includes("amazon")) {
    throw fail("UNSUPPORTED_PAGE", "unsupported page for browser action", { url: ctx?.url });
  }
}

export class BrowserActionContract {
  constructor(name) {
    this.name = name;
  }

  validateContext(_ctx) {
    throw fail("NOT_IMPLEMENTED", `${this.name}.validateContext is not implemented`);
  }

  prepare(_ctx, _payload) {
    throw fail("NOT_IMPLEMENTED", `${this.name}.prepare is not implemented`);
  }

  execute(_ctx, _preparedPayload) {
    throw fail("NOT_IMPLEMENTED", `${this.name}.execute is not implemented`);
  }

  verify(_ctx, _preparedPayload, _executeResult) {
    throw fail("NOT_IMPLEMENTED", `${this.name}.verify is not implemented`);
  }

  summarize(executionResult) {
    return executionResult.summary || `${this.name} executed`;
  }
}

export async function runContractAction(action, ctx, payload) {
  const startedAt = Date.now();

  action.validateContext(ctx);
  const preparedPayload = await action.prepare(ctx, payload);
  const executeResult = await action.execute(ctx, preparedPayload);
  await action.verify(ctx, preparedPayload, executeResult);

  const finishedAt = Date.now();
  return {
    accepted: true,
    action_name: action.name,
    execution_id: `bx_${action.name}_${finishedAt}`,
    duration_ms: finishedAt - startedAt,
    channel: "browser_fallback",
    summary: action.summarize(executeResult),
    result: executeResult
  };
}

export function createMockSuccessResult(actionName, payload) {
  return {
    status: "success",
    action_name: actionName,
    payload,
    observed_at: new Date().toISOString(),
    proof: {
      mode: "mock_browser",
      page: payload.page_hint || "advertising_console"
    }
  };
}
