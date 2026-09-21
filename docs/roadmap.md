# Roadmap, constitution phases vs repo status

Constitution: `docs/APP_CONSTITUTION.md`. Build order (§92): X import →
local vault → normalize → index → search → related → evidence-backed
synthesis → rediscovery. Everything else feeds that loop.

| Phase | Constitution goal | Status |
|---|---|---|
| 1 Local capture | local DB, capture, item storage, basic UI (§7-§10, §39) | API capture + quick-save box + extension; no local-first client queue yet |
| 2 Ingestion | URL/PDF parse, metadata, normalize (§9, §17) | Done: upload + bookmark import + MV3 extension + file serving (open originals, ownership-checked); OCR explicitly later (§17) |
| 2b Sources (NEW) | X first-class + SourceConnector abstraction (§11-§14, §49) | **Next:** `sources` table + XConnector/BrowserConnector + `x` source_type + disconnect keep/delete (§53) |
| 3 Search | FTS, filters, ranking, embeddings (§21-§23) | Hybrid ready: FTS + pgvector column + RRF fusion + lazy vectors; scoped search (in:/topic:/view:) with chips; live vectors wait on `EMBED_KEY` |
| 4 Context | entities, topics, related, dedupe (§32-§35, §47) | Done v1: folder/domain/mined graph with capped confidence + refresh-on-enrich, same-topic relations, topic chips; entity-entity relations + LLM extractor + dedupe open |
| 5 Synthesis | retrieval pipeline, citations, uncertainty (§24-§31) | v1 done: FTS top-12 → Groq prose → code citation gate → per-claim Jev verdicts with source mapping → run log |
| 6 Sync | auth, encrypted sync, conflicts (§39-§40) | **Started:** `sync_state` on items (`local/pending/synced/conflict/failed/deleted`); full auth/conflict UI still open |
| 7 Rediscovery | resurfacing, temporal relevance (§36-§38) | v1 done: old+connected surfacing with reasons, open-event logging started; Quiet/Balanced/Proactive control added (§38) |
| 8 Monetization | billing, BYOK, tiers (§82-§83) | Keys plumbed (BYOK-ready); no billing |
| 9 Platform | API, MCP (§63-§65) | Capture/search API exists; no MCP |
| 10 Agents | suggest → permission → act (§65, App. J) | Explicitly later; no code |

## Next, in order (§92 build order)

1. X import path (V1 gap): connect → backfill → sync (§11-§14).
2. Recent + Archive views (§3, §19): Recent captures + All/Sources/Topics/People/Projects/Collections.
3. Nav to constitution: Search · Recent · Capture · Archive · Settings (§3, §55-§56).
4. Golden dataset before ranking changes (`eval.md`, seeded: `eval/seed-queries.md` + runner, baseline recall@3 = 1.0).
5. Embeddings only when FTS proves insufficient (§44-§46: deterministic first).

Out of V1 entirely (§85): teams, marketplace, agents, graph viz, gamification,
companions, chat-first. The MVP success test (§80) is the gate:
"I forgot I saved this, but the system found it."
