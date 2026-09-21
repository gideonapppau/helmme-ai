# Test routine, run after every iteration

Takes about 10 minutes. Half is automatic, half is clicking through.

## 1. Automatic (run these, all must pass)

From `services/api`:
```
go test -count=1 ./...
```
From `apps/web`:
```
npx tsc --noEmit
node --test "..\..\packages\context-engine\src\jev.test.ts" "..\..\packages\context-engine\src\policy.test.ts"
node --test "..\extension\lib.test.js"
powershell -ExecutionPolicy Bypass -File docs\eval\run-eval.ps1
```
Last line must read `20 passed, 0 failed`. If it doesn't, the iteration
is not done, fix first, demo later.

## 2. Click-through (5 minutes, in the browser)

1. Open the app. The home screen loads with no errors.
2. Save a note with Quick save. It says "Saved."
3. Search one word from that note. It appears with a bolded excerpt.
4. Click "related" under it. Reasons read like plain words.
5. Ask "What did I learn about X?" → Summarize. Sources are numbered.
6. Click a [n] marker in the summary. It glides to that source and lights it up. Bad markers stay plain text.
7. Tap a claim to expand its sources inline. Claims with none stay plain.
8. Open Cleanup. Groups read plainly; deleting one copy leaves the others.
6. Save a file. It says "Saved."
7. Delete something (via API for now). It disappears from search.
8. Every word on screen is plain. No ports, no internals, no jargon.

## 3. If something breaks

Write down: what you clicked, what you expected, what happened.
One screenshot beats three paragraphs. Fix, then re-run section 1.
