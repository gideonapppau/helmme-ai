# APP CONSTITUTION, Personal Context App (v2.0, 21 Sep 2026)

> Replaces all prior constitutions (`ARCHITECTURE.md`, `spec/product-spec-v1.0.md`,
> `jev-decision-layer.md`). This file is the single source of truth.
> Spec § numbers below refer to sections in this file (§1-§92 + Jev appendix).

## §1 One sentence

**Save anything. Find what you meant later.**

The system turns everything a person saves into a searchable, connected,
evidence-backed personal context layer without requiring manual organization.
The user never thinks "where should I put this?", only "I might need this later. Save it."

## §2 The loop

```
Capture → Store → Understand → Connect → Forget → Retrieve → Synthesize → Use → Capture more
```

Value compounds with accumulation. New users get value from importing existing
information; old users get value from years of context. Accumulation is the moat.

## §3 Visible product (keep it small)

Primary nav only: **Search · Recent · Capture · Archive · Settings.**
Desktop: Search dominates. Mobile: Search + Capture dominate.
No permanent AI-chat screen. No dashboard. No twenty nav items.

## §4 First launch (action, not tutorial)

> Your information is scattered. Save it here. We'll make sense of it later.

Primary: **Start**. Secondary: **I already have things to import.**
Option A (fresh): enter app → "What do you want to remember?" + Capture.
Option B (bring context): connect X / bookmarks / Notes / files / PDFs / Markdown.
Copy: "You already have a memory. Bring it here."

## §5 Account (local-first)

1. Launch → local vault created. 2. Capture → search works offline.
3. Only when user wants sync/another device → create account → encrypted sync.
Account gives: encrypted sync, devices, backup, integrations, recovery, later API access.

## §6 Home (intentionally boring)

Center: **Search your memory** + examples
("What did I save about PostgreSQL?", "Find everything about Corsa Cloud").
Below: **Recent captures** (chronological, 3-5 items).
Optional quiet resurfacing (max 1-2): "You saved this six months ago…" + View.
No endless feed.

## §7-§10 Capture (Save → Done)

No mandatory metadata/title/folder/tags/category. System handles it async.
- **Text:** large field, Save → "Saved." Async enrichment.
- **URL:** store URL + time + source + local ID immediately; fetch/title/text/
  author/date/canonical/metadata/entities/topics/embedding/relations/index async.
- **Share sheet (mobile):** Save to Memory → Save → return to app. Heavy work later.

## §11-§14 X is a first-class V1 source

Settings → Connections → X → "Bring your X bookmarks into your memory." →
OAuth (never password) → "We found N bookmarks. Import them?" (default: all).
Import runs in background with progress (643 / 2,184), user can leave.
Each post becomes a normal item (text, author, handle, post date, URL,
import date, refs, links, media, hashtags). **No X silo**, X posts answer
"What have I saved about PostgreSQL?" alongside notes/articles/PDFs.
Source stays visible as provenance. Continuous sync: check → dedupe → store →
enrich → connect → index. Unavailable upstream items are marked, not silently dropped.

## §15-§18 Other sources (input channels, not products)

- **Browser bookmarks:** URL/title/folder/timestamp imported; folders are
  historical metadata, not organization.
- **Apple Notes:** each note an item; original authoritative; system adds
  topics/entities/concepts/relations/embeddings/dupes, never rewrites.
- **Files (PDF/MD/TXT/DOCX first):** store original → text → OCR → sections →
  embeddings/entities/topics/relations/index. Original stays available.
- **Screenshots/images (V1.5+):** store → OCR → vision → text/entities/
  concepts/embeddings/relations. Image is authoritative.

## §19 Archive (views, not folders)

All · Recently added · Sources · Topics · People · Projects · Collections.
Views over one context graph, never containers.

## §20 Item page

Header (title/source/date/type) → Original content → About (topics/entities) →
Related (+ "Why is this related?") → Actions (edit/remove/hide-relation/
add-to-collection/export/view-source). Derived info visually distinct.

## §21-§23 Search (primary interface, one omnibar)

Exact (`postgres`), semantic ("connection management"), temporal ("last year"),
source ("X posts about…"), person ("from Naval"), project ("Corsa Cloud"),
concept ("ideas about cheaper infra").
Result: title/source/date/excerpt/topics/entities/relation indicator.
Ranking fuses lexical + semantic + entity + topic + temporal + source +
graph proximity + authorship + importance + history + intent, never raw vector distance.

## §24-§31 Synthesis (research doc, not chat)

Search asks "What is here?"; synthesis asks "What does it all tell me?"
Intent detection → explicit prompt ("Synthesize what you've learned about X") → Enter.
Pipeline: intent → retrieval → rerank → graph expansion → evidence select →
compress → model → citation validation → answer.
Screen: header + "Based on N items (18 X · 9 articles…)" + structured sections
(themes, what changed, open questions, contradictions). Every claim cited [1][4];
click opens source. "Why did you say this?" shows direct evidence vs
interpretation vs inference. Comfortable with "3 items, not enough for a
reliable conclusion." Contradictions shown side-by-side with dates; never pick a winner.
Knowledge evolution ("How has my thinking changed?") uses timestamps + notes + concepts.

## §32 Graph (invisible)

Nodes: items/people/orgs/places/products/concepts/topics/projects/events/claims/sources.
Edges: related-to/supports/contradicts/duplicate-of/same-topic/project/person/place/
follow-up/earlier-version/derived-from. User feels "these are connected", never sees a viz.

## §33-§35 Projects / Topics / Collections (emergent, not manual)

- **Projects** emerge from clusters (e.g. VPS+Postgres+pricing+Ghana → "Possible
  project: Corsa Cloud"); user confirms; living view, not folder.
- **Topics** auto-generated (PostgreSQL 23 items…); user can rename/hide/merge/reject,
  never maintain.
- **Collections = saved queries** ("X posts about Corsa Cloud"), dynamically update.

## §36-§38 Rediscovery + Why this + Quiet

Resurface only when current research connects to old saves, with reason and
actions (Open / Save to project / Not relevant / Why this?).
"Why this?" always explains shared concepts. Default **Quiet**; Balanced /
Proactive are behavior controls, not personalities.

## §39-§40 Offline + Sync

Offline capture: local ID → save locally → pending → confirm now, sync + enrich later.
Item sync states: `local · pending · synced · conflict · failed · deleted`.
User content never silently lost on conflict.

## §41-§43 Privacy + Raw vs Derived

User owns context: inspect/edit/export/delete/reject-relations/remove-derived/
disable-enrichment/sync/integrations/revoke-devices.
Raw layer (what user captured) always wins over derived layer (what system thinks).
Derived facts versioned: claim + confidence + source ids + model + version + timestamps.

## §44-§46 Providers + Pipeline + Progressive enrichment

Capability router, never one-model lock-in: deterministic/cheap for metadata/
classification/entities; efficient for topics/relations/summaries; frontier only
for synthesis/contradictions/reasoning.
Queue: `ITEM_CREATED → extract_metadata → extract_text → OCR → classify →
entities → topics → embedding → dedupes → relations → graph → index.`
Jobs async/idempotent/retryable/observable/versioned.
Bulk imports: Phase 1 store → 2 deterministic metadata → 3 index → 4 recent/
high-value → 5 older progressively. Searchable before enrichment finishes.

## §47-§48 Duplicates + External links

Same article from X/bookmarks/notes → one primary + "saved from" history.
X post → references → Article preserved as edge; article becomes own memory if imported.

## §49 SourceConnector (one abstraction for every integration)

```
authorize() backfill() sync() normalize() reconcile() disconnect()
→ XConnector, BrowserConnector, AppleNotesConnector, RedditConnector,
  YouTubeConnector, GmailConnector, DriveConnector …
```

## §50-§53 Settings / Export / Delete / Disconnect

Settings groups: Account (profile/devices/security) · Connections · Privacy
(sync/AI/retention/providers) · Memory (enrichment/resurface/relations) ·
Models (default/BYOK/local) · Data (export/import/delete).
Export: everything or subset (items/project/collection/source) as Markdown/JSON/
originals (later SQLite). Never trap the user.
Delete: "Delete this memory?" removes raw + derived per retention policy; user
never thinks about embeddings/index/edges.
Disconnect: "Keep imported memories" vs "Delete imported memories", disconnect
never destroys context unless chosen.

## §54 Filters (secondary)

Source/date/type/topic/person/project/collection chips. Natural language first.

## §55-§58 Nav + Keyboard + Global capture

- Mobile: Search · Recent · Capture (floating) · Archive · Settings. Search opens to keyboard.
- Desktop sidebar: Search · Recent · Archive, Projects, Topics, Settings. Quiet.
- Keys: `Cmd/Ctrl+K` search · `Cmd/Ctrl+Shift+S` capture · arrows navigate ·
  Enter open · `Cmd/Ctrl+Enter` synthesize · Esc close.
- Desktop global shortcut → tiny window → paste → Enter → Saved → disappears.

## §59 Notifications (rare or nothing)

OK: import complete; found a connection; conflicting info exists.
Never: greetings, streaks, "you haven't used the app", engagement gimmicks.

## §60-§62 Current-context search + Decision history + Evolution

"Find what I've saved relevant to what I'm working on now" uses recent
searches/project/captures/clusters. "Why did I decide Postgres?" reconstructs
from notes/articles/timestamps, separating "you wrote…" from "system inferred…".
"What have I changed my mind about?" surfaces old→new positions + contradictions.

## §63-§65 Context API / MCP / Agents (later, permissioned)

```
AI app → Context API → Permission layer → Personal context → Evidence
search_context · retrieve_item · get_topic · get_project · find_related · request_synthesis
```
External AI gets only what is permitted. Agents only after trust:
Understand → Suggest → Ask permission → Act → Record. No autonomy by default.

## §66 Security boundary

Imported content is untrusted data, never instructions. Isolate user
instructions / imported content / system instructions. LLM is never the
authz boundary, authz lives in code.

## §67-§70 Backend

```
Mobile/Desktop/Web → API → Modular Go Backend → Postgres + Queue + Object storage
                                          → Context graph + Processing workers
Local: App → Encrypted local DB (SQLite) → Local search index → Capture queue
```
Server Postgres entities: users, devices, sources, items, item_metadata,
entities, topics, relationships, derived_facts, synthesis_runs, queries,
collections, projects, model_providers, permissions, sync_records, processing_jobs.
Large files → encrypted object storage; DB holds references.

## §71-§74 Data models

- **Item:** id, user_id, source(_id), type, title, content, source_url,
  source_type, created/ captured/updated_at, content_hash, status, visibility,
  raw_storage_reference, sync_state, version. Sources: manual/x/browser/
  apple_notes/reddit/youtube/file/share_sheet.
- **Relationship:** source/target item, type, confidence, evidence, timestamps, model_version.
- **DerivedFact:** claim, confidence, source ids, status (active/uncertain/
  rejected/superseded), model/version, timestamps.
- **SynthesisRun:** query, source ids, model, prompt_version, result, citations,
  created_at, every answer reproducible.

## §75-§76 Omnibar intelligence

`postgres` → Search; "What did I learn…" → Synthesis; "…last year" → Filtered
retrieval; "How is X connected to Y?" → Relation exploration. One box handles
Search/Filter/Retrieve/Compare/Connect/Synthesize/Explore.

## §77-§79 Empty / First session / Killer demo

Empty: "Your memory starts here." + Save something + Bring what you have.
First session: connect X → 500+ bookmarks → search "startup ideas" →
"what patterns do I keep saving?" → citations → forgotten find. That is activation.
Demo: "I have 2,000 bookmarks I never re-read" → import → "What have I collected
about making money from software?" → old posts+notes+articles+projects →
"Which matter for what I'm building now?" Two queries prove the thesis.

## §80-§81 Activation + Retention

Activated = captured/imported several → retrieved one → inspected evidence →
forgotten item proved useful. North star: **Useful Context Retrieved.**
Loop: Save → Forget → Need → Search → Find → Trust → Save more → Richer archive → Repeat.
No daily-engagement hacks.

## §82-§83 Monetization

Charge for intelligence, not storage/notes.
Free: local capture, basic search, limited sync/enrichment/size.
Plus (~$8-15/mo, validate): sync, semantic search, enrichment, synthesis,
resurface, imports, larger archive. BYOK: user key, pay for software/infra.
Pro (later): bigger archive, integrations, limits, API, MCP, tools.

## §84-§85 V1 vs Not-V1

V1: capture (text/URL/files/share) · sources (manual + X) · local-first vault +
sync foundation · understanding (metadata/topics/entities/embeddings/relations/
dupes) · retrieval (keyword/semantic/filters) · synthesis (grounded/cited/
uncertain) · context (related/projects/topics) · data (export/delete/disconnect).
NOT V1: social, teams, marketplace, avatar/personality, graph viz, agents, tasks,
CRM, calendar, email client, PM suite, dashboards, gamification/streaks, dozens
of integrations, automation builder, custom training.

## §86-§88 Later

V1.5: extension, Apple Notes, Reddit, YouTube, screenshots/OCR, project +
contradiction detection, better resurface, global capture, BYOK.
V2: context API, MCP, Gmail/Drive, AI-chat imports, decision history, evolution,
collections, cross-source reasoning. V3: personal context infrastructure, other apps ask what user knows/projects/decisions/relevant-now.

## §89-§90 Architecture diagram + Layers

USER → Capture+Connect → Vault → Raw → Understanding → Entities+Topics →
Graph → Retrieve/Connect/Resurface → Synthesis → Evidence → User → Action → More context.
Layers: 1 Capture · 2 Memory (store safely) · 3 Understanding (meaning) ·
4 Context (connections) · 5 Intelligence (retrieve/synthesize/resurface/expose).
User feels L5; moat is L2-L4.

## §91 The product rule (non-negotiable)

**Never make the user work harder than the information is worth.**
`title → folder → tag → project → category → description → confirm` = failure.
**Save.** Then the machine works.

## §92 Final UX

"I saw something useful." Save. "I forgot." Months later: "What was that thing
about X?" → found. "What else do I know?" → connected. "What does it all tell
me?" → synthesized. "Why are you saying that?" → evidence. "What have I
forgotten that matters?" → resurfaced. Not another place to maintain info, the layer that makes accumulated info useful.
**Build order: X import → local vault → normalize → index → search → related →
evidence-backed synthesis → rediscovery.** Everything else feeds that loop.

---

# Appendix J, Jev: decision/action layer on top of memory (separate brand)

**Memory answers "What do I know?" Jev answers "Given what I know, what should I do next?"**
Do not merge into one indistinguishable assistant-notes-tasks-memory-agent blob.
Memory stays "Save anything. Find what you meant later." Jev stays
"Turn context into better decisions and action." Shared infra, distinct UX.

- Memory = long-term store; Jev = reasoning over it; Corsa apps = execution.
- Jev retrieves via permissioned Context API only:
  `search_context · retrieve_item · find_related · get_project ·
  get_decision_history · request_synthesis`. Never sees unscoped archive.
- Jev writes back durable context (decisions with Situation → Options →
  Evidence → Constraints → Assumptions → Decision → Outcome → Review + citations),
  so "Why did we target agencies?" is answerable months later.
- Jev must distinguish "you explicitly wrote…" from "system inferred…".
- Jev never generates open prose as judge: it returns choices/scores/booleans
  with probabilities (query-router 7 intents, memory-state 10 states,
  source-classifier, relationship, claim-verifier); code thresholds decide;
  text models write prose from verified claims only. All calls logged to
  decision_events with model version + odds.
- Sequence: Phase 1 memory engine → Phase 2 Jev against it → Phase 3 context API
  → Phase 4 other apps/agents. Jev is the first major consumer of the context
  infrastructure, not a feature inside the capture box.
- Jev's advantage is persistent evidence-backed user context, not prompts.
  Understand → Suggest → Ask permission → Act → Record. No autonomy by default.
