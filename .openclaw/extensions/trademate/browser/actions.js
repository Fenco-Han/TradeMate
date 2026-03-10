import {
  BrowserActionContract,
  createMockSuccessResult,
  runContractAction,
  validateRelayContext
} from "./action_contract.js";

function requiredString(payload, key) {
  const value = payload?.[key];
  if (typeof value !== "string" || value.trim() === "") {
    const err = new Error(`${key} is required`);
    err.code = "INVALID_PAYLOAD";
    throw err;
  }
  return value.trim();
}

class PauseCampaignAction extends BrowserActionContract {
  constructor() {
    super("pause_campaign");
  }

  validateContext(ctx) {
    validateRelayContext(ctx);
  }

  prepare(_ctx, payload) {
    return {
      campaign_id: requiredString(payload, "campaign_id"),
      reason: payload?.reason || "fallback_pause"
    };
  }

  execute(_ctx, preparedPayload) {
    return createMockSuccessResult(this.name, preparedPayload);
  }

  verify(_ctx, preparedPayload, executeResult) {
    if (executeResult.status !== "success") {
      throw Object.assign(new Error("pause campaign failed"), { code: "VERIFY_FAILED" });
    }
    if (!executeResult.payload?.campaign_id || executeResult.payload.campaign_id !== preparedPayload.campaign_id) {
      throw Object.assign(new Error("pause campaign verify failed"), { code: "VERIFY_FAILED" });
    }
  }

  summarize() {
    return "campaign paused via browser fallback";
  }
}

class ResumeCampaignAction extends BrowserActionContract {
  constructor() {
    super("resume_campaign");
  }

  validateContext(ctx) {
    validateRelayContext(ctx);
  }

  prepare(_ctx, payload) {
    return {
      campaign_id: requiredString(payload, "campaign_id"),
      reason: payload?.reason || "fallback_resume"
    };
  }

  execute(_ctx, preparedPayload) {
    return createMockSuccessResult(this.name, preparedPayload);
  }

  verify(_ctx, preparedPayload, executeResult) {
    if (executeResult.status !== "success" || executeResult.payload?.campaign_id !== preparedPayload.campaign_id) {
      throw Object.assign(new Error("resume campaign verify failed"), { code: "VERIFY_FAILED" });
    }
  }

  summarize() {
    return "campaign resumed via browser fallback";
  }
}

class AddNegativeKeywordAction extends BrowserActionContract {
  constructor() {
    super("add_negative_keyword");
  }

  validateContext(ctx) {
    validateRelayContext(ctx);
  }

  prepare(_ctx, payload) {
    return {
      campaign_id: requiredString(payload, "campaign_id"),
      ad_group_id: requiredString(payload, "ad_group_id"),
      keyword_text: requiredString(payload, "keyword_text"),
      match_type: requiredString(payload, "match_type")
    };
  }

  execute(_ctx, preparedPayload) {
    return createMockSuccessResult(this.name, preparedPayload);
  }

  verify(_ctx, preparedPayload, executeResult) {
    if (executeResult.status !== "success" || executeResult.payload?.keyword_text !== preparedPayload.keyword_text) {
      throw Object.assign(new Error("add negative keyword verify failed"), { code: "VERIFY_FAILED" });
    }
  }

  summarize() {
    return "negative keyword added via browser fallback";
  }
}

class PauseKeywordAction extends BrowserActionContract {
  constructor() {
    super("pause_keyword");
  }

  validateContext(ctx) {
    validateRelayContext(ctx);
  }

  prepare(_ctx, payload) {
    return {
      keyword_id: requiredString(payload, "keyword_id"),
      campaign_id: requiredString(payload, "campaign_id")
    };
  }

  execute(_ctx, preparedPayload) {
    return createMockSuccessResult(this.name, preparedPayload);
  }

  verify(_ctx, preparedPayload, executeResult) {
    if (executeResult.status !== "success" || executeResult.payload?.keyword_id !== preparedPayload.keyword_id) {
      throw Object.assign(new Error("pause keyword verify failed"), { code: "VERIFY_FAILED" });
    }
  }

  summarize() {
    return "keyword paused via browser fallback";
  }
}

const ACTIONS = {
  pause_campaign: new PauseCampaignAction(),
  resume_campaign: new ResumeCampaignAction(),
  add_negative_keyword: new AddNegativeKeywordAction(),
  pause_keyword: new PauseKeywordAction()
};

export function listSupportedActions() {
  return Object.keys(ACTIONS);
}

export async function runBrowserAction(actionName, context, payload) {
  const action = ACTIONS[actionName];
  if (!action) {
    const err = new Error(`unsupported action: ${actionName}`);
    err.code = "UNSUPPORTED_ACTION";
    throw err;
  }

  return runContractAction(action, context, payload);
}
