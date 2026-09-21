# PROGRESS, what was done, in plain words

> Updated every work session. No jargon: each entry says what changed,
> which files were touched, why, and how it was checked.

## How to read this
- **Done** = built and checked. **Checked** tells you how (live = ran against the
  real database/API; tests = automated checks; static = careful reading, no live run).
- Plan source: `docs/BUILD_PLAN.md`. Rules source: `docs/APP_CONSTITUTION.md`.
  Movement source: `docs/APP_FLOWS.md`.

---

## Session 1, 21 Sep 2026: new rules + X-first backend + web nav

### 1. New app constitution (replaced the old one)
- **Why:** you pasted a new master document and said the old rules no longer apply.
- **Files:** created `docs/APP_CONSTITUTION.md`; deleted `docs/ARCHITECTURE.md`,
  `docs/spec/product-spec-v1.0.md`, `docs/jev-decision-layer.md`, `docs/spec/`;
  updated `docs/README.md`, `docs/roadmap.md`, `README.md` to point at the new file.
- **Result:** one rulebook (§1-§92 + Jev appendix). Checked by reading it back fully.

### 2. Build plan (ordered task list)
- **Why:** you asked for an incremental plan so tasks get checked one after another.
- **Files:** created `docs/BUILD_PLAN.md` (updated live as tasks complete).
- **Result:** 17 phases, hierarchical, each with a pass/fail check.

### 3. Flows + UX contract + skills
- **Why:** you asked that the app's movement (data + user journeys on mobile/desktop/web)
  and the "stupid simple but beautiful" standard be written down, and that all needed
  skills be found and installed.
- **Files:** created `docs/APP_FLOWS.md`. Installed into `.agents/skills/` (recorded in
  `skills-lock.json`): `frontend-design`, `web-design-guidelines`,
  `vercel-react-best-practices`, `vercel-react-native-skills`, `e2e-testing-patterns`.
  Skipped ecosystem PDF skills (too few users to trust; local `pdf` skill covers it)
  and one stale index entry (repo had no such skill).
- **Result:** movement spec + UX rules + skill list, all in the codebase.

### 4. X as a first-class source (backend)
- **Why:** the constitution says X bookmarks are a V1 source and must become normal
  searchable memory, not a separate pile, through one shared connector shape.
- **Files:** created `db/migrations/017_sources_sync.sql` (sources list + sync states),
  `services/api/sources.go` (connectors, connect/import/disconnect, recent + archive views);
  edited `services/api/main.go` (allow `x` + `share_sheet` saves, register routes).
- **Checked live:** connect → import (2 in, 1 saved + 1 honest duplicate) → search finds the
  X post → recent + archive show it → test data removed, archive re-checked clean.

### 5. X login scaffold (OAuth) + import worker
- **Why:** real X login needs keys you don't have yet, so the full shape was built with
  honest "not configured" answers instead of fake success.
- **Files:** created `db/migrations/019_x_oauth.sql` (server-side token store + import progress),
  `services/api/xoauth.go` (login start/callback, token exchange, logout revoke, background
  import worker), `services/api/xoauth_test.go` (automatic checks incl. an official PKCE test);
  edited `docs/env.md` (where to put keys when you have them).
- **Result:** without keys every route says `x_not_configured` (checked live, status 503).
  One test caught a typo in my own test answer; fixed after checking the official RFC.

### 6. Database fixes found by audit
- **Why:** reading all 19 database files found two ways a fresh setup would break.
- **Files:** created `db/migrations/018_pgcrypto.sql` (fresh databases failed their first save
  without this); edited `dev.cmd` (was starting the wrong database image and loading only
  the first of 19 files, now correct image + all files in order, stopping on any error).
- **Checked live:** all 19 files applied to a throwaway database with zero errors; 17 tables
  verified; same files applied to the dev database; throwaway deleted.

### 7. Web navigation to the constitution
- **Why:** the constitution allows only Search · Recent · Capture · Archive · Settings.
- **Files:** edited `apps/web/app/page.tsx` (nav on desktop + mobile, Recent + Archive panels,
  X-connect box, quiet/balanced/proactive control, keyboard shortcuts, new home words).
  Also fixed a pre-existing crash (`BrushIcon` was used but never defined).
- **Checked:** type check clean. Click-through in the browser is still open (for your return).

## Open items waiting on you (updated end of Session 3)
1. **Browser run-through** (`http://localhost:3001`): Recent/Archive panels, X-connect box, mood buttons,
   arrow-key walk, Esc closes all. API underneath is live-verified; this is eyes-on-yours.
2. **X keys** (`X_CLIENT_ID`, `X_CLIENT_SECRET`, `X_REDIRECT_URL` in `docs/env.md`), unlocks the real
   login round-trip + bookmark import + continuous sync.
3. **Model keys** (`GROQ_API_KEY` for synthesis prose, `EMBED_KEY` for meaning-vectors), unlocks
   enrichment (topics/entities), prose synthesis, and the reject-path live test.
4. **Fresh-boot run** of updated `dev.cmd` on a clean machine (exercises 001→023 + pgvector image).

## Loop status
- Phase 0: done (0.1-0.4, live-verified). Phase 1: 1.1-1.5 scaffolded + live-verified except real-keys parts (1.4 round-trip, 1.5 live run, 1.6 sync).
- Next up: Phase 2, offline-proof capture (see BUILD_PLAN).

## Session 2, 21 Sep 2026: the loop runs (all live-verified)

### 8. Offline-proof capture (Phase 2.1)
- **Why:** a phone with no signal must save now and sync later, with the real time kept and no twins on retry.
- **Files:** created `db/migrations/020_offline_capture.sql` (your-own-id + no-twins rule),
  `services/api/capture_test.go` (automatic checks); edited `services/api/main.go` (accepts
  `client_id` + `captured_at`, same id twice returns the first save).
- **Checked live:** same save sent twice → same id + `duplicate:true`, 1 row; real time kept
  (08:00, not sync time); bad dates get 400. Test rows removed. (One live failure on the way:
  my timestamp pointed at the wrong slot, fixed, re-checked green.)

### 9. Article body reading (Phase 3.2)
- **Why:** a saved link was only findable by its title/address. Now its real sentences are filed too.
- **Files:** edited `services/api/fetch.go` (reads the article part, skips menus/buttons/code,
  files the author + publish date, ignores non-pages); extended `services/api/fetch_test.go`.
- **Checked live:** saved `https://example.com` → title + body sentence filed → body words
  searchable → test removed. File serving (200, ownership-checked) and the safety guard
  (block-list proven by tests) marked verified with it.

### 10. Search judgments logged + own-notes-first (Phase 4.2-4.4)
- **Why:** the router's calls must be auditable, and "what did I decide" must put your words first.
- **Files:** created `services/api/decisions.go` (judgment log, only known judges allowed),
  `services/api/ranking_test.go`; edited `services/api/search.go` (own-thinking nudge),
  `apps/web/app/api/route-query/route.ts` (files each judgment, never blocks the answer).
- **Checked live:** judgment stored + id returned, unknown judge → 400; on the real archive,
  "what did I write about helmme" puts your own note above the URL and PDF. Test rows removed.

### 11. Projects as living views (Phase 5.3)
- **Why:** projects must emerge from what you save, confirming one files nothing, deleting one destroys nothing.
- **Files:** created `db/migrations/021_projects.sql`, `services/api/projects.go`
  (list/suggest/declare/confirm/rename/remove), `services/api/projects_test.go`;
  edited `services/api/main.go` (routes) and the archive `projects` view in `services/api/sources.go`
  (now reads the projects table with live counts).
- **Checked live:** declared "Corsa Cloud" → confirmed with live count → archive shows it →
  removed view-only → archive empty again. (PowerShell lesson: `$pid` is reserved, used a real name.)
- Still open: suggestions need a richer archive (topics with 3+ saves); topic rename/hide/merge has no endpoint yet (reject exists).

## Session 3, 21 Sep 2026: the loop runs to the gates (all live-verified)

### 12. Server moods for resurfacing (Phase 7.3)
- **Why:** Quiet/Balanced/Proactive was a button that changed nothing. Now the server enforces it.
- **Files:** created `db/migrations/022_settings.sql` (one saved mood per user, Quiet default),
  `services/api/settings.go` (read/save mood, only the three known moods),
  `services/api/settings_test.go`; edited `services/api/resurface.go` (caps results 1/3/5 by mood),
  `services/api/main.go` (routes), `apps/web/app/page.tsx` (Settings reads + saves the server mood).
- **Checked live:** default `quiet` → set `proactive` stored → bad mood 400 → reset to `quiet`.
  Full cap proof (1 vs 5 showing) waits on an archive with 5+ candidates.

### 13. Arrow-key results + panel fixes (Phase 8.3)
- **Why:** keyboard users were stuck: arrows did nothing and Esc left two panels open.
- **Files:** edited `apps/web/app/page.tsx` (up/down walks result links, typing never hijacked,
  Enter opens natively, Esc now closes everything).
- **Checked:** type check clean. Browser run-through is on your return list (no browser here).

### 14. Takeout/export (Phase 10.2) + disconnect proof (10.4)
- **Why:** your archive must be yours, one download carries everything, never secrets.
- **Files:** created `services/api/export.go` (JSON/Markdown, optional source filter, sources table
  never read), `services/api/export_test.go` (reads well + leak test); edited `services/api/main.go`.
- **Checked live:** real archive exports in both formats; bad format/source → 400; fixed live that
  note text echoed into the `url` field. Disconnect keep/delete proven in Session 1.

### 15. Delete tombstones (Phase 10.3)
- **Why:** delete must mean delete, but other devices must later learn it was on purpose.
- **Files:** created `db/migrations/023_delete_log.sql` (death record with no ties, survives the delete);
  edited `services/api/delete.go` (writes it before removing anything).
- **Checked live:** create → delete → item 0 rows, contents 0 rows, tombstone 1 row.

### 16. Collections, duplicates, reasons, eval (Phase 5.5, 5.6, 7.2, 12.1)
- **Checked live, no new code needed:** saved search shows live counts + `new`, opening stamps it,
  deleting removes it; duplicates endpoint correctly empty; resurface carries its reason;
  **golden eval re-run after all today's search changes: 20 passed, 0 failed, recall@3 = 1.0.**

## Session 4, 21 Sep 2026: the loop resumes, no keys needed (all live-verified)

You said don't stop, so nothing below waited on keys.

### 17. Guesses moved out of your bytes (Phase 10.5)
- **Why:** the fetcher was scribbling its guesses (author, date) inside your original save.
  Now originals stay byte-identical; guesses live beside them with their source attached.
- **Files:** created `db/migrations/024_item_metadata.sql` (guesses table + one-time lift of old keys);
  `services/api/metadata_test.go` (live-DB proof); edited `services/api/fetch.go` (writes guesses aside),
  `services/api/item.go` (serves them flagged as `derived`).
- **Checked live:** guess filed with source, original untouched, guesses die with the item.
  Reject path proven on a real entity (removed + remembered, then fully restored, archive verified identical after).

### 18. Linked saves meet (Phase 5.7) + source filter (8.5) + current-context lift (7.4)
- **Why:** a note pasting a saved article's link should meet that article; typing `source:x`
  should equal tapping the X chip; places you've read lately rise a little.
- **Files:** edited `services/api/related.go` (`referenced` group, strongest first),
  `services/api/search.go` (`source:` scope incl. literal names, recent-domain lift),
  `services/api/ranking_test.go`, `apps/web/app/page.tsx` (X chip).
- **Checked live:** manufactured link pair met with `referenced` first (then removed);
  `source:url` returns only urls (fixed live: literal names first fell through as search words).
  Eval re-run after both ranking changes: still 20/20.

### 19. Text-file uploads + open (Phase 9.3)
- **Why:** notes and markdown are saves too, they ride with PDFs and images now.
- **Files:** edited `services/api/upload.go` (sniff-checked .md/.txt, capped, no asset row),
  `services/api/files.go` (serves them via a locked second door), `apps/web/app/page.tsx` (picker accepts them);
  created `services/api/upload_text_test.go`.
- **Checked live:** upload → searchable → open (200) → removed. DOCX still open (needs a new library).

### 20. CI gates (Phase 12.2, first half)
- **Why:** the machine must refuse bad releases by itself.
- **Files:** created `.github/workflows/ci.yml` (format + vet + tests + type check on every push/PR).
  Golden eval stays manual, it needs the real archive.

### 21. Housekeeping done right
- Your own X connection row (`@gideonapppau`) was left connected; only my stale test counters were zeroed.
- Your pre-existing corrections (2 rows) untouched. Sweep after every test: zero test rows left.

### 22. Icons: every glyph is a real icon now (UI polish)
- **Why:** you were right, icon hygiene had slipped. The back link was a typed `←`,
  "open original" a typed `↗`, close/remove buttons a typed `×`, the Connect X button
  wore a chain-link icon, and X saves got a generic file tile. Typed punctuation is
  not an icon system.
- **Files:** edited `apps/web/lib/ui.tsx` (new filled `XLogoGlyph`, the true X mark,
  not Hugeicons' thin stroke approximation; `x` added to `TYPE_STYLES` so X saves get
  their own tile), `apps/web/app/globals.css` (X tile colors, light + dark),
  `apps/web/app/page.tsx` (Connect X wears the X mark; Settings X card gets an X tile
  anchor + mark on the button; scope-chip and saved-view close buttons use `Cancel01Icon`;
  Review now uses `ClipboardCheckIcon` so it stops sharing Recent's `HistoryIcon`;
  unused imports removed), `apps/web/app/item/[id]/page.tsx` (new `BackLink` with
  `ArrowLeft02Icon` used on all three page states; `open original`/`open file` carry
  `ArrowUpRight01Icon`; topic/entity remove buttons use `Cancel01Icon`; the header chip
  is now the real colored tile + plain-word label).
- **Checked:** `tsc --noEmit` clean. Production build compiled successfully with lint +
  types green (the `_app.js` rejection during page-data collection is a pre-existing
  Next 15 pages-router probe, unrelated). Icon names verified against the installed
  `@hugeicons/core-free-icons` package before use.

### 23. Typography: four voices, each with one job (UI polish)
- **Why:** Inter is the most default interface font on the internet, the last
  "generated page" tell in the system. And the app had no reading voice: the moments
  a person actually reads (a saved note, extracted PDF prose, Jev's synthesis answer)
  rendered in the same 13px interface sans as everything else.
- **Files:** edited `apps/web/app/layout.tsx` (Hanken Grotesk replaces Inter for the
  interface, warm humanist grotesque that fits the paper; Source Serif 4 joins as the
  fourth voice), `apps/web/app/globals.css` (sans chain + new `--font-serif` /
  `font-serif` voice), `apps/web/app/page.tsx` (synthesis answers now read in serif
  14.5px/1.75 with the citation taps kept; rail stamps unified 9.5px/0.18em → 10px/0.16em
  to match every other stamp), `apps/web/app/item/[id]/page.tsx` (saved notes read in
  serif 15.5px/1.85, extracted PDF prose in serif 14.5px/1.8).
- **Voice map:** Hanken Grotesk = interface · Outfit = wordmark · Geist Mono =
  archival stamps (time, domains, counts, eyebrows) · Source Serif 4 = words you read.
- **Checked:** `tsc --noEmit` clean. Clean `.next` production build exit 0, compiled,
  lint + types green, all pages generated; Hanken Grotesk @font-face and the serif
  voice confirmed present in the compiled CSS.

## Session 5, 21 Sep 2026: Word files closed (live-verified)

### 23. DOCX support (Phase 9.3 complete)
- **Why:** the last V1 file type. Word files were refused; now their words are saved and searchable.
- **Files:** `go.mod`/`go.sum` (MIT `nguyenthenguyen/docx`, picked after reading its license + API in the
  module cache, it returns raw XML, so text runs are pulled out with a small matcher);
  edited `services/api/upload.go` (zip-sniffed, capped, filed as generic `file` so search filters stay clean),
  `services/api/files.go` (same locked door serves them), `apps/web/app/page.tsx` (picker + words);
  extended `services/api/upload_text_test.go` (builds a minimal Word file in-test, no fixtures).
- **Checked live:** upload → searchable ("zebra migration quarterly") → open (200) → removed.
  Eval after: still 20/20. (Caught by tests on the way: the library needs the links file present,
  and returns raw XML, both handled, both covered by tests.)

## Session 6, 21 Sep 2026: your GROQ key was already wired (live-verified)

### 24. Full synthesis with real prose + real Jev verdicts (Phase 6.1)
- **Why:** the pipeline had only ever run in fallback mode (evidence, no prose). Your env already
  carried the keys, so the real path got proven.
- **Files:** none, code already existed (`synth.go`, `claims.go`). Verified, not built.
- **Checked live:** "what have I saved about infrastructure" → prose with 3 citations, gate passes,
  5 genuine Jev claim verdicts, run #17 logged. Key inventory: `GROQ_API_KEY` present,
  `TYPESAFE_API_KEY` present (verdicts are real judgments), `EMBED_KEY` missing (vectors off by design).

## Session 7 — 21 Sep 2026: plain writing sweep (verified)

### 25. No em dashes, no slop, simple words everywhere
- **Why:** you asked for plain writing across the repo.
- **Files:** 51 files touched (`docs`, `services/api`, `apps/web`, `apps/extension`).
  Rules: spaced em dash to comma, wrapped-line dash joins on, leftovers to hyphen,
  en dash to hyphen. Slop-word check came back empty (none of moreover/furthermore/
  leverage/seamless/robust/delve/crucial and kin existed). Arrows, middots, and
  section marks kept (they carry meaning, checked at byte level).
- **Checked:** dash recount zero, `go vet` + full tests green, `tsc` clean, API rebuilt
  healthy, eval still 20/20.

### 26. Connected memories: a detached thread, not a card grid (UI polish)
- **Why:** you asked for the related cards to feel detached, linked like a dream, not
  packed in a grid. Direction drawn from the obsidianui.dev / rareui.com language.
- **Files:** edited `apps/web/app/item/[id]/page.tsx` (rebuilt "Connected memories" as
  a vertical thread: a hairline that fades in from the words and dissolves at the end,
  one ringed link-node per memory sitting on the thread, a tick reaching from node to
  card, staggered rise-in. Hover on a card tints its node and tick ember; first attempt
  put `group` on the inner Link where it could not reach the node, caught and fixed.
  Grid replaced by the single column of hanging memories). "Open original"/"open file"
  are now link-button pills with `LinkSquare02Icon` (was bare text + arrow).
- **Also:** the Archive empty state now hosts the Rare UI `Folder` component (already
  in the codebase at `components/ui/folder-component.tsx`, small scale, white theme),
  with honest words under it.
- **Checked:** `tsc --noEmit` clean; production build exit 0, all pages generated.

## Session 8, 21 Sep 2026: the loop keeps going, no keys needed (all live-verified)

### 26. Topic controls (Phase 5.4)
- **Why:** machine guesses need human overrule: rename, hide, merge.
- **Files:** created `db/migrations/025_topic_controls.sql`, `services/api/topics.go`, `services/api/topics_test.go`;
  edited `services/api/main.go` + hidden respected in `search.go`, `related.go`, `resurface.go`, `projects.go`.
- **Checked live:** real folder import made 2 topics, renamed one, hid one (gone from lists, kept with flag),
  merged (links moved, count 4), cleaned to baseline.

### 27. Enrichment switch (Phase 10.1)
- **Why:** your data, your call: guesses off must never break saving or searching.
- **Files:** created `db/migrations/026_enrichment_toggle.sql`; edited `services/api/settings.go` (both knobs
  optional and separate), `services/api/graph.go` (single choke point in `enrichItem`).
- **Checked live:** off, save, 0 entities; on, save, 3 entities; both removed; switch left on.

### 28. Usage metrics (Phase 12.3)
- **Why:** the north star is finding, not collecting. Now it is counted.
- **Files:** created `db/migrations/027_metrics.sql`, `services/api/metrics.go`; edited `services/api/main.go`
  (search logging that never breaks the answer).
- **Checked live:** probe search moved the counters; page shows searches, hits, syntheses, opens, captures.

### 29. Merge proof (Phase 5.6)
- **Why:** duplicates must fuse and unfold without loss.
- **Files:** none, code already existed. Verified, not built.
- **Checked live:** manufactured twins grouped (same-file + same-link), merged to 1 visible, unmerged to 2, removed.

### 30. Extension gaps closed (Phase 14.1)
- **Why:** highlights and notes were stored but unfindable; no shortcut; no note box; no image menu.
- **Files:** edited `services/api/main.go` (quoted words join the index, capped) + `quoted_test.go`;
  `apps/extension/lib.js` + `lib.test.js` (note param), `popup.html`/`popup.js` (note box),
  `manifest.json` (shortcut + background), new `background.js` (image menu).
- **Checked live:** highlight words and note words both searchable; extension tests 5/5.
  Browser proof of popup/menu still open (no browser here).

### 31. Stored edges (Phase 5.2)
- **Why:** every "why this?" deserves a stored row with kind, confidence, evidence, stamps.
- **Files:** created `db/migrations/028_relationships.sql`, `services/api/edges.go`,
  `services/api/edges_test.go`; edited `services/api/graph.go` (hook), `services/api/main.go` (route).
- **Checked live:** same-site edge filed with evidence, read back with stamps, cascade wiped on delete.
  Related ranking untouched, eval still 20/20.

## Loop status (end of Session 8)
- Newly done + live-verified: 5.4 (topic controls) · enrichment switch · 12.3 (metrics) · 5.6 (merge proof) ·
  14.1 (extension gaps) · 5.2 (stored edges). Eval still 20/20 after every ranking change.
- Still genuinely gated (BUILD_PLAN "Held items"): X round-trip/import/sync (your X keys, application
  in progress, use-case text drafted in chat) · bulk enrichment incl. `EMBED_KEY` (optional; FTS carries us) ·
  share sheet/account/desktop/mobile (client builds) · full proofs/demo (richer archive) ·
  billing/platform/agents (later phases).
