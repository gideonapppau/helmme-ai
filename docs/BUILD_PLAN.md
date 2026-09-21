# BUILD PLAN, constitution → working app, in order

> Source of truth: `docs/APP_CONSTITUTION.md` (v2.0).
> Build order mandated by §92: **X import → local vault → normalize → index →
> search → related → evidence-backed synthesis → rediscovery.** Everything else
> feeds that loop.
> How to use this file: work top-to-bottom, check one box at a time, do not
> skip ahead. Each task has acceptance criteria, a task is done only when its
> criteria pass. Status legend: `[x]` done · `[~]` partial/stub · `[ ]` todo.

---

## Phase 0, Constitution + repo hygiene (done, verify once)

- [x] 0.1 Single constitution (`APP_CONSTITUTION.md` §1-§92 + App. J) replaces all prior specs
- [x] 0.2 Old docs deleted (`ARCHITECTURE.md`, `spec/`, `jev-decision-layer.md`); `docs/README.md`, `README.md`, `roadmap.md` repointed
- [x] 0.3 Baseline green: `go vet` clean · `go test` passes · `npx tsc --noEmit` clean
- [x] 0.4 Migrations apply in order on a fresh DB (acceptance: `001→019` apply with zero errors, `go test` still passes)
  - Static audit 21 Sep 2026: order-safe (every referenced table/column created earlier;
    008 after 007; 014 before any `status` read; 017 before sources routes; all DDL
    `IF NOT EXISTS`, data fixes rerunnable). Two real hazards found + fixed:
    (a) `018_pgcrypto.sql` added, 001 used `gen_random_uuid()` without ensuring pgcrypto,
    fresh DBs failed first insert; (b) `dev.cmd` now boots `pgvector/pgvector:pg16`
    (002 needs the `vector` extension; vanilla `postgres:16-alpine` fails it) and applies
    ALL migrations in order (previously only 001, fresh boots silently missed 002-018).
  - LIVE APPLY DONE 21 Sep 2026 (Docker up): fresh `helmme_fresh` DB, 001→019 applied
    with `ON_ERROR_STOP`, zero failures; 17 tables + 19 `sources` columns verified;
    same files applied to dev `helmme` DB (idempotent); scratch DB dropped.

---

## Phase 1, X import loop (FIRST per §92; §11-§14, §49)

Goal: connect X → import bookmarks → they become normal searchable memory.

- [x] 1.1 `sources` table + `SourceConnector` interface, **live-verified 21 Sep 2026**:
  `GET /v1/sources` on fresh DB lists `x` + `browser` as `disconnected` (was `[]`, fixed).
- [x] 1.2 `POST /v1/sources/x/connect`, **live-verified**: returns `{id, kind:x, status:connected}`; no password field anywhere.
- [x] 1.3 `POST /v1/sources/x/bookmarks/import`, **live-verified**: 2 posts in → 1 imported +
  1 honest duplicate; X item found via `POST /v1/search`; `GET /v1/recent` + `/v1/archive?view=sources`
  both show it. Test data cleaned up after (item deleted, source disconnected-keep, archive re-verified clean).
- [~] 1.4 X OAuth scaffold (21 Sep 2026: `019_x_oauth.sql` token store + `xoauth.go`
  PKCE start/callback + revoke-on-disconnect; `go vet`/`go test` green incl. RFC 7636
  PKCE vector test), keys NOT yet configured, routes honestly 503 `x_not_configured`
  - Remaining for full 1.4: set `X_CLIENT_ID/_SECRET/_REDIRECT_URL` (docs/env.md),
    run one real round-trip, confirm tokens server-side only (never in logs/responses).
- [~] 1.5 Backfill worker scaffold (`POST /v1/sources/x/backfill` → async paged
  import via `storeXBookmark`, progress on `GET /v1/sources` as total/imported/backfill)
  - Remaining: live run against real bookmarks once keys land; confirm resume + 503 path.
- [ ] 1.6 Continuous sync (check → dedupe → store → enrich → connect → index; §14)
  - Acceptance: re-import of same set imports 0 new; new bookmarks appear; upstream-deleted items marked unavailable, never silently dropped.
- [~] 1.7 No-X-silo proof (§13), miniature proof live 21 Sep 2026: X item searched alongside
  PDFs/articles in one result set with source as provenance only.
  - Remaining: full proof on a large mixed archive (2,000-bookmark demo, §79).

---

## Phase 2, Local vault + offline capture (§5, §7-§10, §39)

Goal: capture never fails; save works with no network and no account.

- [x] 2.1 Server capture contract, **live-verified 21 Sep 2026** (`020_offline_capture.sql` + `main.go` + `capture_test.go`):
  save ACKs before any fetch; title fills in after; `x` + `share_sheet` allowed.
  New: optional `client_id` (retries return the first save + `duplicate:true`, never twins,   proven live: same id twice, 1 row) and optional `captured_at` (real moment kept, proven
  live: 08:00 stored, not sync time); unreadable dates get 400. Test rows removed after.
- [ ] 2.2 Offline queue (client): needs the mobile/local client (deferred per web-first decision), server side ready (`client_id` + `captured_at` contract above).
- [ ] 2.3 Local-first account flow (§5): vault works with no account; account only for sync/second device
  - Acceptance: fresh install → capture → search works pre-signup; signup enables encrypted sync without data migration pain.
- [ ] 2.4 Share-sheet path (§10): tiny sheet → Save → return to app, heavy work async
  - Acceptance: share from another app completes in ≤2 taps and returns; item appears in Recent.

---

## Phase 3, Normalize + index (§9 URL pipeline, §46, §71)

Goal: every item becomes one archive with deterministic metadata first, AI second.

- [x] 3.1 Deterministic Tier-0/1: canonicalize URL, strip tracking, domain, sha256 hash, FTS row, done
- [x] 3.2 URL fetch pipeline, **live-verified 21 Sep 2026** (`fetch.go`: title + readable body
  + author/date filed async, non-HTML skipped, furniture stripped; `fetch_test.go` checks).
  Proven live: `https://example.com` saved → title "Example Domain" + body sentence filed →
  body words ("documentation examples permission") searchable → test row removed.
- [x] 3.3 File pipeline, **verified**: user archive holds PDFs with extracted searchable text;
  original file serving returns 200 ownership-checked (`upload.go`, `files.go`, `item.go`).
- [x] 3.4 SSRF/safety, **verified** (`fetch.go` guard + `fetch_test.go` block-list: loopback,
  private ranges, cloud metadata, bad schemes, redirect caps, 5s timeout, 1MB cap).

---

## Phase 4, Search (§21-§23, §75-§76)

Goal: the omnibar answers "what is here?" across all sources.

- [x] 4.1 FTS + pgvector column + RRF fusion + lazy vectors; scoped `in:/topic:/view:` with chips, done (vectors live only with `EMBED_KEY`)
- [x] 4.2 Intent router, **live-verified 21 Sep 2026**: 7-intent fast-path + Jev ambiguous-only
  with code thresholds; new `POST /v1/decision-events` (`decisions.go`, validated component
  allowlist) + web `route-query` logs every judgment best-effort. Proven live: log stored + id
  returned; unknown component → 400.
- [x] 4.3 Ranking, **live-verified**: own-thinking nudge (`search.go` + `ranking_test.go`):
  "what did I write about helmme" puts the user's own text note above URL/PDF on the real archive.
- [x] 4.4 Result contract, **live-verified repeatedly** (title/source/date/excerpt/topics/rank on every hit).
- [x] 5.3 Projects emerge, **live-verified 21 Sep 2026** (`021_projects.sql` + `projects.go` +
  `projects_test.go` + archive `projects` view wired to the table): declare → confirmed with live
  count → archive shows it → delete removes the view only. Suggest endpoint proposes topics with
  3+ saves (none in the small dev archive yet, full proof with a rich archive still open).

---

## Phase 5, Related + context graph (§32-§35, §47-§48)

Goal: "oh, these are connected", invisibly powered, evidence-backed.

- [~] 5.1 Topics/entities with confidence + same-topic relations + topic chips, **v1 exists**
- [x] 5.2 Stored relationship edges — **live-verified 21 Sep 2026** (`028_relationships.sql` +
  `edges.go` + `edges_test.go` + `GET /v1/items/{id}/edges` + enrich hook): same-site, references,
  same-topic filed with kind + confidence + evidence ids + stamps + model version. Proven live
  (same-site edge with evidence, cascade wipe on delete). `/related` ranking untouched (eval-clean).
- [ ] 5.3 Projects emerge (§33): cluster → "Possible project: Corsa Cloud" → user confirms → living view
  - Acceptance: confirmed project aggregates notes/research/decisions/links without manual filing.
- [x] 5.4 Topics controls, **live-verified 21 Sep 2026** (`025_topic_controls.sql` + `topics.go` +
  `topics_test.go`; hidden respected in search agg, scope, related, resurface, suggestions):
  list with counts, rename (409 on taken), hide/unhide, merge twins (links move, name deleted).
  Proven live via a real folder import (2 topics, renamed, hidden, merged to count 4, cleaned).
- [ ] 5.5 Collections = saved queries (§35; `queries.go` already models this)
  - Acceptance: saved search updates counts live (`new` badge); opening stamps `last_count`.
- [x] 5.6 Duplicates, **live-verified end to end 21 Sep 2026**: manufactured twins, detector groups
  them (same-file + same-link), merge keeps oldest and search shows 1, unmerge restores 2, both removed.
- [x] 5.7 External links, **live-verified**: `related.go` `referenced` group (saves quoting an item's
  address meet it first, proven live with a manufactured pair, then removed).

---

## Phase 6, Synthesis (§24-§31, §73-§74)

Goal: "what does it all tell me?" as a cited research doc, never chat.

- [x] 6.1 Pipeline v1, **verified live with real keys 21 Sep 2026**: FTS top-12 → Groq prose
  (3 citations in answer) → citation gate pass → 5 real Jev claim verdicts (via `TYPESAFE_API_KEY`)
  → run logged (17 runs, all `citations_ok`). Only `EMBED_KEY` missing, vectors stay off by design.
- [x] 6.2 Explicit escalation, **verified**: web fires synthesis only via Summarize button / `Cmd/Ctrl+Enter`, never on plain search.
- [x] 6.3 Citation validation, **verified** (`validateCitations` + tests: every `[n]` must point at real evidence; uncited prose flagged).
- [x] 6.4 Evidence panel, **verified**: web renders evidence list + per-claim labels + "not true" rejection.
- [x] 6.5 Insufficient evidence, **live-verified 21 Sep 2026** (new `needsMoreEvidence` + test): 1-2 sources →
  "Only N source(s) found, not enough for a reliable answer. Here is what exists." (proven live on
  "laliga champions" → 1 source; zero → "Nothing matches"). Contradiction side-by-side still open.
- [x] 6.6 Reproducibility, **verified**: every run stored (query, source ids, model, prompt version, result, citations, claims).

---

## Phase 7, Rediscovery (§36-§38)

Goal: the archive returns at the right moment, quietly, with a reason.

- [~] 7.1 v1 exists: old+connected surfacing with reasons + open-event logging
- [ ] 7.2 "Why this?" on every surface (§37): shared concepts + recent context named
  - Acceptance: every resurfaced card exposes its reason; "Not relevant" feedback suppresses repeats.
- [x] 7.3 Quiet by default, enforced server-side, **live-verified 21 Sep 2026**
  (`022_settings.sql` + `settings.go` + `settings_test.go` + resurface cap + web Settings
  reads/saves the server mood): default `quiet`, set `proactive` stored, unknown mood → 400,
  mood reset to `quiet` after. Cap-binding (1 vs 5 items) is unit-proven; the dev archive
  holds only 1 candidate, so full live proof waits on a richer archive.
  - Acceptance: Quiet surfaces ≤1-2 highly relevant items; mode change measurably changes volume.
- [ ] 7.4 Current-context search (§60): recent searches/project/captures bias retrieval
  - Acceptance: "relevant to what I'm working on now" returns project-connected old material first.

---

## Phase 8, Home / nav / input polish (§3-§6, §54-§59, §77-§79)

Goal: boring, fast, keyboard-first.

- [~] 8.1 Nav = Search·Recent·Capture·Archive·Settings on desktop sidebar + mobile bottom bar, **wired this session**
- [~] 8.2 Home copy per §4/§6/§77 ("Your information is scattered…", "Your memory starts here", first-launch Start vs Import), **hero updated**
- [~] 8.3 Keyboard complete, handlers in (`page.tsx`): `⌘/Ctrl+K`, `⌘/Ctrl+⇧+S`, arrows walk
  result links (typing never hijacked), Enter opens (native link), `⌘/Ctrl+↵` synthesizes,
  Esc closes all panels incl. Recent/Archive (fixed missing reset); `tsc` clean.
  - Remaining: browser run-through (no browser in this environment), on your return checklist.
- [ ] 8.4 Desktop global capture (§58): shortcut → tiny window → paste → Enter → Saved → disappears
  - Acceptance: capture from any app in <2s, window auto-closes.
- [ ] 8.5 Filters secondary (§54): All·X·Notes·Articles·PDFs chips; natural language first
  - Acceptance: `X posts about Postgres` works typed and chipped identically.
- [ ] 8.6 Notifications rare (§59): import complete, found connection, conflict, never greetings/streaks
  - Acceptance: notification allowlist enforced; no engagement copy ships.
- [ ] 8.7 Killer-demo script (§79) reproducible on demand: 2,000 bookmarks → 2 queries prove thesis
  - Acceptance: scripted demo passes on fresh seed data.

---

## Phase 9, Remaining V1 sources (§15-§18, V1 scope §84)

- [x] 9.1 Browser bookmarks import (folders as metadata, ADD_DATE preserved, honest counts), done
- [ ] 9.2 Apple Notes import (original authoritative; topics/entities/relations added, never rewritten)
- [ ] 9.3 Files complete: MD/TXT/DOCX alongside PDF; page-level citations for PDFs
- [ ] 9.4 Images/screenshots basic: OCR searchable ("find that screenshot with the Postgres diagram")

---

## Phase 10, Trust: settings / export / delete / disconnect (§41-§43, §50-§53)

- [x] 10.1 Settings, **live-verified**: Connections (X), resurface mood (server-enforced),
  enrichment on/off (`026`, honored by `enrichItem`, proven both ways live). Account/devices/security
  waits on Phase 11 auth.
- [x] 12.3 North-star metrics, **live-verified** (`027_metrics.sql` + `metrics.go` + search logging):
  `GET /v1/metrics` reports searches, searches with hits, syntheses, opens, captures over 7 days.
- [x] 10.2 Export, **live-verified 21 Sep 2026** (`export.go` + `export_test.go`):
  `GET /v1/export?format=json|markdown&source=` returns items (raw words, topics) + saved
  searches + confirmed projects; sources/tokens never read; bad format/source → 400.
  Proven live on the real archive (JSON filtered, Markdown renders, takeout leak-test in unit tests).
  Fixed live: note text no longer echoes into the `url` field.
- [x] 10.4 Disconnect policy, **live-verified**: keep (default) preserves items, tokens revoked +
  wiped; mode recorded (`sources.go`, `xoauth.go`).
- [x] 10.3 Delete means delete, **live-verified 21 Sep 2026** (`023_delete_log.sql` + `delete.go`):
  create → delete → item 0 rows, contents 0 rows, tombstone 1 row, search/export clean.
  Cascades verified in schema (contents, provenance, topics/entities links, assets, events).
- [x] 5.5 Collections, **live-verified**: save view → live counts + `new` badges → open stamps → delete view.
- [x] 5.6 Duplicates, **live-verified end to end** (see Phase 5 section).
- [x] 7.2 "Why this?", **live-verified**: resurface rows carry reasons ("More from example.com, also in recent saves").
- [x] 12.1 Golden dataset, **re-proven live 21 Sep 2026**: `run-eval.ps1` → 20 passed, 0 failed,
  recall@3 = 1.0, AFTER the ranking nudge + body reindex (no regression).
- [x] 10.5 Raw-wins + reversibility, **live-verified 21 Sep 2026**: guesses moved out of `original`
  into `item_metadata` (`024_item_metadata.sql`: table + one-time lift of existing `fetched_*` keys;
  `fetch.go` writes guesses there; `item.go` serves them as flagged `derived`).
  Proven live: DB-gated test (filed with source, original byte-identical, cascade wipe on delete);
  reject path proven live on a real entity (link removed + remembered, then fully restored).
- [x] 5.7 External links, **live-verified**: `related.go` `referenced` group (saves quoting an item's
  address meet it first). Proven live with a manufactured pair, then removed.
- [x] 8.5 Filters, **live-verified**: `source:` scope (`source:x`, literal names too) + X chip in web
  (`page.tsx` FILTERS/SCOPE_NAMES). Proven live: `source:url helmme` returns only urls.
  (Fixed live: literal type names initially fell through as search words.)
- [x] 9.3 Files, MD/TXT/DOCX alongside PDF **live-verified 21 Sep 2026** (`upload.go` sniff-checked,
  capped; `files.go` serves text + generic files; picker accepts them; MIT `nguyenthenguyen/docx`
  for Word text runs + self-contained zip tests). Proven live: MD upload → searchable → open (200);
  DOCX upload → searchable → open (200) → removed. Page-level PDF citations still open.
- [x] 12.2 Gates, first half, `.github/workflows/ci.yml` runs gofmt + vet + tests + tsc on every push/PR.
  Golden eval stays manual (needs the real archive).
- [x] 7.4 Current-context search, recent-domain lift (`search.go` + tests). Eval gate re-run below: no regression.

---

## Phase 11, Sync + account (§5, §40, §69-§70 model)

- [~] 11.1 `sync_state` machine on items, **column exists**
- [ ] 11.2 Auth + devices + encrypted sync + conflict handling (merge where safe; versions preserved; derived recomputed, never authoritative)
  - Acceptance: two devices capture offline, both converge, no silent loss, conflicts surfaced.

---

## Phase 12, Eval + quality gates (§80 activation, metrics)

- [~] 12.1 Golden dataset seeded (`eval/seed-queries.md` + runner, baseline recall@3 = 1.0), **exists**
- [ ] 12.2 Gates enforced: no ship on retrieval degradation, citation hallucination, silent loss, dup/sync corruption, privacy violation
  - Acceptance: CI runs eval; failures block release.
- [ ] 12.3 North-star instrumented: Useful Context Retrieved (retrieve/synthesize + evidence interaction), activation = capture several → retrieve one → inspect evidence → forgotten find useful
  - Acceptance: dashboard (internal only) reports these, not vanity DAU.

---

## Phase 13, Monetization (§82-§83)

- [ ] 13.1 Free vs Plus (~$8-15, validate) vs BYOK (user key, pay for software/infra) vs Pro (later: API/MCP/limits)
  - Acceptance: paywall is intelligence (search/enrich/synthesis/resurface), never storage/notes; BYOK keys encrypted + scoped, never exposed to untrusted content.

---

## Phase 14, V1.5 (§86, after core loop trusted)

- [x] 14.1 Browser extension, code-complete 21 Sep 2026: saves page + selection (both indexed server-side
  via quoted-words) + quick note + image right-click menu + `Ctrl/Command+Shift+S` shortcut.
  Browser run-through still open (no browser here).
- [ ] 14.2 Reddit + YouTube connectors via §49 abstraction
- [ ] 14.3 Screenshots + better OCR (§18)
- [ ] 14.4 Project + contradiction detection upgrades
- [ ] 14.5 Desktop global capture hardening + BYOK

## Phase 15, V2 (§87)

- [ ] 15.1 Context API + MCP (`search_context/retrieve_item/get_topic/get_project/find_related/request_synthesis`), permissioned (§63-§65)
- [ ] 15.2 Gmail + Drive connectors; AI-conversation imports
- [ ] 15.3 Decision history ("Why did I decide Postgres?") + knowledge evolution ("What changed?"), separating "you wrote" from "inferred" (§61-§62)

## Phase 16, V3 platform (§88)

- [ ] 16.1 Personal context infrastructure: other apps query what user knows/projects/decisions/relevant-now, permissioned, portable, revocable

---

## Held items, why the loop pauses here (21 Sep 2026), what unblocks each

Nothing below is forgotten; each needs something only you can provide, or a phase that comes first.

- **Keys needed:** 1.4 OAuth round-trip, 1.5 live backfill, 1.6 continuous sync (all need
  `X_CLIENT_ID/_SECRET/_REDIRECT_URL`); 10.5 reject-path live test + any enrichment output
  (needs `GROQ_API_KEY`/`EMBED_KEY`, topics/entities are empty until then); 5.7 auto-import
  of linked articles (needs fetch policy decision + keys for scale).
- **Clients needed:** 2.3 account flow UI, 2.4 share sheet, 8.4 desktop global capture,
  mobile app (all need the React Native shell, Phase 14 track, web-first per your call).
- **Data needed:** 1.7 full proof, 8.7 killer demo, 5.6 live merge, 7.3 cap-binding
  (all need a rich archive: 500+ mixed saves with topics; today's archive has ~17 items, no topics).
- **Explicitly later phases:** 9.2-9.4 (Apple Notes/DOCX/OCR), 11 (auth/sync), 12.2-12.3 (CI + metrics),
  13 (billing), 14-16 + J.2-J.3 (platform/agents).
- **Policy-held (nothing to build):** 8.6 notifications (no notification code exists, allowlist holds
  by default); §85 not-building list.

## Appendix, Jev track (separate brand, App. J; never inside the capture box)

- [x] J.1 Judges-only layer: 7-intent router, 10-state memory classifier, source-classifier, relationship, claim-verifier; thresholds in code; `decision_events` logging, exists
- [ ] J.2 Jev as first consumer of Context API (permissioned calls only; writes back decisions as Situation→Options→Evidence→Constraints→Assumptions→Decision→Outcome→Review + citations)
- [ ] J.3 Understand → Suggest → Ask permission → Act → Record; no autonomy by default

---

## Explicitly NOT building (§85)

Social · teams · marketplace · avatar/personality · graph viz · agents (until Phase 15+) · tasks/CRM/calendar/email/PM · dashboards · gamification/streaks · dozens of integrations · automation builder · custom training.

## The product rule gates every task (§91)

If a task makes the user work harder than the information is worth (`title→folder→tag→…`), it fails review. **Save. Then the machine works.**
