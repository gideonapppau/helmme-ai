# Roadmap — V0 phases vs repo status

Spec phases: §145–154. MVP scope: §116 (text/URL/PDF →
capture → store → search → related → synthesis with citations).

| Phase | Spec goal | Status |
|---|---|---|
| 1 Local capture | local DB, capture, item storage, basic UI | API capture + quick-save box + extension; no local-first client queue yet |
| 2 Ingestion | URL/PDF parse, metadata, normalize | Done: upload + bookmark import + MV3 extension + file serving (open originals, ownership-checked); OCR explicitly later (§97) |
| 3 Search | FTS, filters, ranking, embeddings | Hybrid ready: FTS + pgvector column + RRF fusion + lazy vectors; scoped search (in:/topic:/view:) with chips; live vectors wait on `EMBED_KEY` |
| 4 Context | entities, topics, related, dedupe | Done v1: folder/domain/mined graph with capped confidence + refresh-on-enrich, same-topic relations, topic chips; entity-entity relations + LLM extractor + dedupe open |
| 5 Synthesis | retrieval pipeline, citations, uncertainty | v1 done: FTS top-12 → Groq prose → code citation gate → per-claim Jev verdicts with source mapping → run log |
| 6 Sync | auth, encrypted sync, conflicts | Not started (§§69–70 design only) |
| 7 Rediscovery | resurfacing, temporal relevance | v1 done: old+connected surfacing with reasons, open-event logging started (revisit filter activates with history) || 8 Monetization | billing, BYOK, tiers | Keys plumbed (BYOK-ready); no billing |
| 9 Platform | API, MCP | Capture/search API exists; no MCP |
| 10 Agents | suggest → permission → act | Explicitly later; no code |

## Next, in order (§112 milestones logic)

1. PDF + image capture path (MVP content gap).
2. Intent taxonomy → spec's 7 intents (§22) in router + UI.
3. Entities/topics/related items (Phase 4) with Jev relationship calls.
4. Synthesis pipeline with citation gate (Phase 5) on one query:
   "What did I learn about X?"
5. Golden dataset before ranking changes (`eval.md`, seeded: `eval/seed-queries.md` + runner, baseline recall@3 = 1.0).
6. Embeddings only when FTS proves insufficient (§136: no new infra early).

Out of V1 entirely: teams, marketplace, agents, graph viz, gamification,
companions, chat-first (§115). The MVP success test (§118) is the gate:
"I forgot I saved this, but the system found it."
