// Pure payload builder — no chrome.* calls, so node --test covers it.
// The popup is glue; this is the contract with POST /v1/items.
function buildItem(tab, selection) {
  const url = (tab && tab.url) || "";
  const title = ((tab && tab.title) || "").trim().slice(0, 500) || url;
  const original = { captured_from: "extension" };
  const sel = (selection || "").trim().slice(0, 5000);
  if (sel) original.selected_text = sel;
  return { source_type: "url", raw_ref: url, title, original };
}

function isSavableUrl(url) {
  return /^https?:\/\//i.test(url || "");
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = { buildItem, isSavableUrl };
}
