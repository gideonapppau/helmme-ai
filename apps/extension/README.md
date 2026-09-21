# helmme extension (Chrome, MV3), local prototype

One button: **Save**. No collections, no tags, no questions. The system
files it later; that is the whole point.

## Load it

1. Make sure the app is running (`dev.cmd` in the repo root, or
   `docker compose up -d`).
2. Open `chrome://extensions`, turn on **Developer mode**.
3. **Load unpacked** → pick `apps/extension`.
4. Pin helmme to the toolbar. Click it on any page → **Save**.

Restricted pages (`chrome://`, the web store) cannot be saved, the
button says so instead of failing quietly.

## What it sends

`POST http://localhost:8081/v1/items` with the page URL, title, any
selected text, and `captured_from: "extension"`. Same validation,
canonicalization, and provenance as every other capture channel.

Point it at production later by changing the one `API` line in
`popup.js`.
