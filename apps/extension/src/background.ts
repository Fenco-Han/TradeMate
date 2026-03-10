chrome.runtime.onInstalled.addListener(() => {
  console.log("TradeMate extension installed");
});

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === "trademate.ping") {
    sendResponse({ ok: true });
  }
});

