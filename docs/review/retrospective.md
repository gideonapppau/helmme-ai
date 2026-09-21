# Retrospective — everything built, planned properly

Date: 2026-09-20. Every shipped slice reviewed: what exists, what covers
it, what is owed. Nothing below is new scope — it is the debt on what
we already claim works.

## 0. Version control (the riskiest item)

- Built: one commit ("Initial commit"). Everything since is uncommitted.
- Owed: per-slice commit history so any slice can be reviewed or
  reverted. Verify `.env.local` is ignored before the first commit.
- Plan: stage by slice (foundation → security → jev → upload → import →
  extension → related → search → synthesis → docs), concise messages,
  then push. Never commit keys, `data/`, or `node_modules`.

## 1. Capture + search foundation

- Built: Item model, POST /v1/items, POST /v1/search (FTS + excerpts +
  de-slugged + list-free understanding), omnibar UI. All live-verified.
- Covered: 17 Go tests, tsc, 18 node tests, migrations 001–006.
- Owed: no DELETE endpoint (raw SQL was used twice — the proof),
  no GET single item, tenant is a hardcoded stub (no auth).
- Plan: DELETE + GET item with ownership checks and tombstone note;
  auth stays stubbed but the seam stays explicit.

## 2. Security posture

- Built: validation, size caps, sniffed uploads, tenant scoping,
  parameterized queries, opaque logs, restrictive CORS.
- Owed: no rate limiting, no auth (localhost-only is the current
  boundary — true today, must not survive first deploy), upload
  endpoint trusts any local caller.
- Plan: rate limits on write endpoints; document "localhost is the
  boundary" in ARCHITECTURE.md; auth design before any network exposure.

## 3. Jev decision layer

- Built: question sets, thresholds in code, mock tests, live
  `/api/route-query` with direct key. Verified ambiguous → human-review.
- Owed: fastPath is TRIPLICATED (page.tsx, route.ts, context-engine) —
  one source, two importers. Router still runs 6 ops vs the spec's 7
  intents (§22). `as any` casts in route.ts. Thresholds (0.6/0.7) are
  guesses with no labeled data. Claim verifier exists but synthesis v1
  does not call it — the flagship safety feature is designed, not built.
- Plan: single-source the router; align to 7 intents; wire per-claim
  Jev checks into synthesis; calibrate thresholds on 50 labeled queries.

## 4. Upload + PDF extraction

- Built: multipart upload, PDF text (stream-order + gap spacing +
  control-char cleaning), image dims, content-addressed storage.
- Owed: extractor artifacts remain (ligature �, merged em-dashes);
  scanned PDFs record `has_text:false` with no OCR path; no endpoint
  serves originals back (uploads are write-only); storage is local-only.
- Plan: file-serving endpoint with auth + signed semantics first
  (uploads must be retrievable); OCR adapter interface second (not the
  implementation); S3 swap only when leaving localhost.

## 5. Bookmark import

- Built: HTML parse (both nest shapes), folders as metadata, ADD_DATE
  time travel, honest counts, live-verified.
- Owed: no other adapters (Apple Notes, read-later); pre-1970 dates
  unhandled; re-import is safe but unexplained in UI.
- Plan: Apple Notes adapter next (spec §53 order); nothing else until
  volume demands it.

## 6. Extension

- Built: MV3 popup, Save → Done, selection capture, restyled to app
  tokens, 4 tests.
- Owed: no icons, no keyboard shortcut, API URL is a hardcoded
  constant, no options page.
- Plan: icons + shortcut + options page for the URL. Small, bounded.

## 7. Related items

- Built: endpoint (saved-together, same-site, similar-words, time
  neighbors) with reason codes, inline UI, merge tests.
- Owed: thin without entities; trigram threshold (0.08) uncalibrated;
  no Jev relationship judgments.
- Plan: calibrate threshold on real archive; Jev relationship calls
  arrive with entities/topics.

## 8. Synthesis v1

- Built: retrieval → Groq prose → code citation gate → run log;
  graceful no-key fallback; UI panel. Live-verified fallback.
- Owed: prose itself unverified live (no key yet); per-claim Jev
  checks missing; `[n]` citations not clickable; no persistent views.
- Plan: key → live prose check → clickable citations → per-claim Jev
  gate → saved views. In that order.

## 9. Copy + UI language

- Built: simple-English pass over app, settings, extension.
- Owed: vigilance only — the rule decays with every new string.
  Rule: no ports, no internals, no milestone names on screen. Ever.

## 10. Docs + toolchain

- Built: spec ingested, roadmap, env, eval plan, teardown, Jev mapping.
- Owed: ARCHITECTURE.md still says "Corsa" (stale vs v1.0 spec);
  no API reference; eval dataset doesn't exist; toolchain rule
  (go directive ≤ Dockerfile) lives in chat, not docs.
- Plan: reconcile ARCHITECTURE.md to v1.0; write toolchain rule down;
  seed golden dataset with 20 queries before next ranking change.
