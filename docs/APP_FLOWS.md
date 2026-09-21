# APP FLOWS, how data moves, how users move, how it must feel

> Companion to `APP_CONSTITUTION.md` (the what/why) and `BUILD_PLAN.md` (the order).
> This file is the exact movement spec: data end-to-end, user journeys per
> platform, and the UX contract. Build every screen against this file.
> Constitution refs in (§).

---

## 1. End-to-end data flow (exact)

```
CAPTURE → STORE → UNDERSTAND → CONNECT → FORGET → RETRIEVE → SYNTHESIZE → USE → CAPTURE MORE
```

### 1.1 Capture (§7-§10, §39)
- User action is always **Save → Done**. No title, folder, tag, category, explanation, ever.
- Text: large field → Save → "Saved." Everything else async.
- URL: store URL + capture time + source + local ID **immediately**, ACK before any
  fetch. Then async: fetch → title → readable text → author → pub date →
  canonical URL → metadata → entities → topics → embedding → relationships → index.
- Share sheet: Share → app → tiny sheet → Save → back to original app. Heavy work later.
- Offline: local ID → save locally → `pending` → confirm now → sync + enrich when online.
- Failure rule: capture never fails visibly. No network, no account, no problem.

### 1.2 Store (§5, §39-§40, §67-§71)
- Local first: `App → Encrypted local DB (SQLite) → Local search index → Capture queue`.
- Server: `API → Modular Go backend → Postgres + Queue + Object storage → Context graph + workers`.
- Every item: id, user_id, source(_id), type, title, content, source_url,
  source_type, created/captured/updated_at, content_hash, status, visibility,
  raw_storage_reference, sync_state (`local·pending·synced·conflict·failed·deleted`), version.
- Original is immutable. Raw layer always beats derived layer. Large files → encrypted
  object storage, DB holds references.

### 1.3 Understand (§44-§46)
- Deterministic first (free): URL, domain, title, author, timestamps, MIME, hash.
- Then cheap models: classification, entities. Efficient models: topics, relations, summaries.
- Frontier models only: synthesis, contradiction analysis, deep reasoning.
- Queue, in order: `ITEM_CREATED → extract_metadata → extract_text → OCR →
  classify → entities → topics → embedding → dedupes → relations → graph → index`.
- Jobs are async, idempotent, retryable, observable, versioned.
- Bulk imports: store everything → deterministic metadata → index → recent/high-value
  first → older progressively. **Searchable before enrichment finishes.**

### 1.4 Connect (§32-§35, §47-§48)
- Graph nodes: items, people, orgs, places, products, concepts, topics, projects,
  events, claims, sources. Edges carry type + confidence + evidence + timestamps.
- Below-bar confidence → tentative or dropped, never presented as fact.
- Duplicates: same article from X/bookmarks/notes → one primary + "saved from" history.
- External links: post → references → article edge preserved.
- The graph is invisible. The user only ever feels "these are connected."

### 1.5 Forget (§2, §81)
- The user forgets. That is the design, not a failure. The retention loop depends on it:
  Save → Forget → Need → Search → Find → Trust → Save more.

### 1.6 Retrieve (§21-§23, §75-§76)
- One omnibar. `postgres` → Search · "What did I learn…" → Synthesis ·
  "…last year" → Filtered retrieval · "How is X connected to Y?" → Relation exploration.
- Ranking fuses lexical + semantic + entity + topic + temporal + source + graph
  proximity + authorship + history + intent. Never raw vector distance.
- Results carry title, source, date, excerpt, topics/entities, relation indicator.

### 1.7 Synthesize (§24-§31)
- Explicit escalation only: results + "Synthesize what you've learned about X" → Enter.
- Pipeline: intent → retrieval → rerank → graph expansion → evidence select →
  compress → model → citation validation → answer.
- Screen is a research doc: header + "Based on N items (18 X · 9 articles…)" +
  themes + what changed + open questions + contradictions. Every claim cited;
  click opens source. "Why did you say this?" separates evidence / interpretation / inference.
- Thin evidence → "not enough for a reliable conclusion." Conflicts shown
  side-by-side with dates; the system never picks a winner.

### 1.8 Use → Capture more (§79-§81)
- Evidence inspected → original opened → related explored → resurfaced item reused →
  decision recorded → more captured. North star: **Useful Context Retrieved.**

---

## 2. User journeys (exact, per platform)

### 2.1 First launch (§4), all platforms
1. "Your information is scattered. Save it here. We'll make sense of it later."
2. Primary **Start** → straight into the app: "What do you want to remember?" + Capture.
   Secondary **I already have things to import** → "You already have a memory. Bring it here."
   → connect X / bookmarks / Notes / files / PDFs / Markdown.
3. No account gate. Local vault created silently. Account only when sync/second device wanted.

### 2.2 First successful session (§78), the activation script
1. Connect X → 500+ bookmarks import (progress `643 / 2,184`, user free to leave).
2. Search "startup ideas" → results appear.
3. "What patterns do I keep saving?" → synthesis with citations.
4. Click citation → exact source opens.
5. Forgotten save discovered. **That moment is activation.**

### 2.3 Killer demo (§79), two queries prove the thesis
1. "I have 2,000 bookmarks I've never re-read." → connect, import.
2. "What have I been collecting about making money from software?" → old posts + notes + articles + projects.
3. "Which of these matter for what I'm building now?" → old context connected to current work.

### 2.4 Mobile journey (§3, §6, §10, §55)
- Bottom bar: **Search · Recent · Capture · Archive · Settings.** Capture floats primary.
- Search opens directly into the keyboard. Capture is ≤2 taps (floating action or share sheet).
- Home: search box → recent captures (3-5, chronological) → at most 1-2 quiet resurfaces.
- Share from any app: Share → Save to Memory → Save → back. Instant return.
- Heavy reading, synthesis, and source inspection work, but capture and search dominate.

### 2.5 Desktop journey (§3, §56-§58)
- Sidebar (quiet): **Search · Recent · Archive, Projects, Topics, Settings.** Search dominates.
- Keys: `Cmd/Ctrl+K` search · `Cmd/Ctrl+Shift+S` capture · arrows move ·
  Enter opens · `Cmd/Ctrl+Enter` synthesizes · Esc closes. Full flow keyboard-only, no traps.
- Global shortcut from anywhere: tiny window → paste → Enter → Saved → disappears (<2s).
- Desktop is where deep work happens: powerful search, synthesis, source inspection, archive management, file import.

### 2.6 Web journey (§3, §6, §19-§20)
- Same five nav items. Search-first omnibar; synthesis is explicit escalation, never default.
- Recent: chronological captures, no ranking. Archive: views (All / Sources / Topics /
  People / Projects / Collections), live counts, never folders.
- Item page: header → original → About (topics/entities) → Related + why →
  edit / remove / hide-relation / add-to-collection / export / view-source.
  Derived info visually distinct from raw.
- Settings: Account · Connections (X connect: handle only, never password) · Privacy ·
  Memory (resurface Quiet/Balanced/Proactive) · Models (BYOK) · Data (export/import/delete).
- Disconnect X: "Keep imported memories" (default) vs "Delete imported memories",   disconnect never destroys context unless chosen.

### 2.7 Rediscovery moment (§36-§38)
- "You saved this six months ago. It may matter for your current Corsa Cloud research
 , it discusses predictable infrastructure pricing." Actions: Open / Save to project /
  Not relevant / Why this? "Why this?" names the shared concepts. Default Quiet.

---

## 3. UX contract, stupid simple, beautiful, functional

Grounded in the constitution (§91-§92: Save, then the machine works) and the
installed design skills (`.agents/skills/frontend-design`, `web-design-guidelines`).

### 3.1 Stupid simple (non-negotiable)
- One box does everything; modes are detected, never selected.
- Capture is one gesture; retrieval is one query; synthesis is one extra Enter.
- No onboarding tutorial, action teaches. Empty states invite ("Your memory starts
  here." + Save something + Bring what you have), never explain.
- Copy in plain verbs, sentence case, user vocabulary ("Save", "Saved.", "Search your
  memory"). Same name for an action everywhere it appears. Errors say what happened
  and how to fix it; they never apologize vaguely.
- Provenance always visible but quiet: source/type/date on everything, derived
  material visually distinct, "why this?" one tap away.

### 3.2 Beautiful (restrained, opinionated)
- Quiet, intelligent, personal, technical-but-not-developer-only, trustworthy, mature.
- Whitespace + typography + hierarchy carry the design. One memorable element per
  surface; everything around it disciplined. Cut decoration that doesn't serve the brief.
- Motion answers actions (open, expand, confirm) and shows what changed. No scattered
  ambient animation, no entrance parade. Reduced motion respected.
- Avoid the generic kit: identical card grids, gradient washes as decoration, tracked-out
  caps eyebrows on every heading, `A · B · C` meta strings, `→` on every link.
  Where the constitution pins a direction, it wins; where it leaves room, choose,   don't default.
- Quality floor, unannounced: responsive to mobile, visible keyboard focus, accessible
  contrast, harmonious palette.

### 3.3 Functional (fast and honest)
- Capture ACKs instantly; search works the moment something is saved; synthesis states
  ("Reading your archive…") beat spinners with no words.
- Never fake it: insufficient evidence says so; contradictions show both sides;
  resurfaces carry reasons; counts are exact (imported vs already-saved vs skipped).
- Silence is a feature: no greetings, streaks, scores, or engagement nudges.
- UI reviews run against the live Vercel Web Interface Guidelines before shipping
  (fetch fresh rules, check `apps/web`, report findings as `file:line`).

### 3.4 Current-UI alignment gaps (handled incrementally via BUILD_PLAN Phase 8)
- [ ] Trim decorative motion/ornament toward §3.2 (one bold element per surface).
- [ ] Verify full keyboard-only flow incl. list navigation (handlers exist; §2.5).
- [ ] Global desktop capture window (§2.5), not built.
- [ ] Offline queue + local vault UI states (§1.1), not built.
- [ ] Real X OAuth + background progress UI (§2.2), stub only.

---

## 4. Skill registry, what is installed and what each is for

Installed locally in `.agents/skills/` (locked in `skills-lock.json`). Global install
is unsupported for these (PromptScript), so project-local is the correct scope.

| Skill | Source / installs | Used for |
|---|---|---|
| `frontend-design` | anthropics/skills · 908K | UX contract §3: intentional, restrained product UI; copy discipline |
| `web-design-guidelines` | vercel-labs/agent-skills · 654K | UI compliance reviews of `apps/web` against live Vercel guidelines |
| `vercel-react-best-practices` | vercel-labs/agent-skills · 732K | Next.js/React implementation quality (web app) |
| `vercel-react-native-skills` | vercel-labs/agent-skills · 218K | Mobile app build (§67 stack: React Native / Expo) |
| `e2e-testing-patterns` | wshobson/agents · 22.8K | Flow-level E2E tests for §2 journeys; Phase 12 eval gates |
| `typesafe-ai` | pre-existing local | Jev decision-layer calls (`experimental_evaluate`, Appendix J) |
| `pdf` / `docx` / `xlsx` / `pptx` | pre-existing local | File ingestion pipelines (§17: PDF/DOCX first) |

Deliberately NOT installed: ecosystem PDF skills (all <500 installs, below the 1K
trust bar; local `pdf` skill covers it); `bobmatnyc/...@playwright-e2e-testing`
(index entry is stale, repo contains no such skill; replaced by `e2e-testing-patterns`).
