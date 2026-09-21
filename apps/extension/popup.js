// Popup glue: tab info + selected text in, one POST out. No decisions asked.
const API = "http://localhost:8081";

const titleEl = document.getElementById("title");
const domainEl = document.getElementById("domain");
const selEl = document.getElementById("sel");
const noteEl = document.getElementById("note");
const saveBtn = document.getElementById("save");
const msgEl = document.getElementById("msg");

let tab = null;
let selection = "";

function say(text, cls) {
  msgEl.textContent = text;
  msgEl.className = "msg" + (cls ? " " + cls : "");
}

async function init() {
  try {
    const [t] = await chrome.tabs.query({ active: true, currentWindow: true });
    tab = t || null;
  } catch {
    tab = null;
  }
  if (!tab || !isSavableUrl(tab.url)) {
    titleEl.textContent = "Nothing to save here";
    domainEl.innerHTML = "&nbsp;";
    saveBtn.disabled = true;
    say("Open a web page first.");
    return;
  }
  titleEl.textContent = tab.title || tab.url;
  try {
    domainEl.textContent = new URL(tab.url).hostname;
  } catch {
    domainEl.innerHTML = "&nbsp;";
  }
  try {
    const [res] = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: () => (window.getSelection ? String(window.getSelection()) : ""),
    });
    selection = (res && res.result) || "";
  } catch {
    selection = ""; // restricted pages (chrome://, store) — save without it
  }
  if (selection.trim()) {
    selEl.textContent = selection.trim().slice(0, 160);
    selEl.hidden = false;
  }
}

saveBtn.addEventListener("click", async () => {
  saveBtn.disabled = true;
  say("Saving...");
  try {
    const r = await fetch(`${API}/v1/items`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(buildItem(tab, selection, noteEl ? noteEl.value : "")),
    });
    if (!r.ok) throw new Error(await r.text());
    say("Saved.", "ok");
  } catch {
    say("Could not save. Is the app running?", "err");
    saveBtn.disabled = false;
  }
});

init();
