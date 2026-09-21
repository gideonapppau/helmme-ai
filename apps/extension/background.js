// Background worker: right-click any image, save it. Plain words: the menu
// offers, one click files the image address as a normal save, the page does
// the rest later. No browser check here can prove the menu renders; that
// waits on a real browser run.
const API = "http://localhost:8081";

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: "helmme-save-image",
    title: "Save image to helmme",
    contexts: ["image"],
  });
});

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  if (info.menuItemId !== "helmme-save-image" || !info.srcUrl) return;
  try {
    await fetch(`${API}/v1/items`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        source_type: "url",
        raw_ref: info.srcUrl,
        title: (tab && tab.title ? tab.title + " (image)" : info.srcUrl).slice(0, 500),
        original: { captured_from: "extension-image" },
      }),
    });
  } catch {
    // No surface here to report on; the popup shows API trouble instead.
  }
});
