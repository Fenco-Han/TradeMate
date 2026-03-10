const SUPPORTED_HOST_PATTERNS = [
  "advertising.amazon.",
  "sellercentral.amazon.",
  "amazon.com"
];

function normalizeContext(input = {}) {
  const tabID = Number.isInteger(input.tab_id) ? input.tab_id : -1;
  const url = typeof input.url === "string" ? input.url : "";
  const browser = typeof input.browser === "string" ? input.browser : "chrome";
  const relayAttached = input.attached === true;

  return {
    tab_id: tabID,
    url,
    browser,
    attached: relayAttached
  };
}

function isSupportedHost(url) {
  if (!url) {
    return false;
  }

  return SUPPORTED_HOST_PATTERNS.some((part) => url.includes(part));
}

export function evaluateRelayStatus(input = {}) {
  const ctx = normalizeContext(input);
  const supportedPage = isSupportedHost(ctx.url);
  const attached = ctx.attached && ctx.tab_id > 0 && supportedPage;

  return {
    attached,
    browser: ctx.browser,
    tab_id: ctx.tab_id,
    url: ctx.url,
    supported_page: supportedPage,
    checked_at: new Date().toISOString(),
    error_code: attached ? "" : inferRelayError(ctx, supportedPage)
  };
}

function inferRelayError(ctx, supportedPage) {
  if (ctx.tab_id <= 0) {
    return "RELAY_NOT_ATTACHED";
  }
  if (!supportedPage) {
    return "UNSUPPORTED_PAGE";
  }
  if (!ctx.attached) {
    return "RELAY_NOT_ATTACHED";
  }
  return "UNKNOWN_ERROR";
}

export function buildAttachResult(input = {}) {
  const status = evaluateRelayStatus({ ...input, attached: true });
  return {
    accepted: status.supported_page,
    relay_status: status,
    message: status.supported_page ? "relay attached" : "unsupported page"
  };
}
