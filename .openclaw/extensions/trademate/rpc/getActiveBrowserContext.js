export function handleGetActiveBrowserContext(params = {}) {
  return {
    browser: typeof params.browser === "string" ? params.browser : "chrome",
    tab_id: Number.isInteger(params.tab_id) ? params.tab_id : -1,
    url: typeof params.url === "string" ? params.url : "",
    title: typeof params.title === "string" ? params.title : "",
    collected_at: new Date().toISOString()
  };
}
