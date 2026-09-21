# Corsa §119 invariants — engineering source of truth (abridged).
# Full spec: Desktop `# Personal Memory Platform.txt` (v1.0, 23 Aug 2026).

1. Capture never depends on AI. Capture works offline. ACK immediately, enrich later.
2. Original is immutable. AI output is never original truth. Every inference has provenance.
3. Every Item has durable identity (canonical_item_id / occurrence / snapshot).
4. Progressive enrichment only: Tier0 always, Tier1 cheap, Tier2 selective, Tier3 justified.
5. AI router chooses capabilities, not providers. Every external call logs processing ledger.
6. Untrusted content is sandboxed. LLM is never the security boundary. Authz in code.
7. Sensitive data has explicit policy. Privacy modes gate cloud calls.
8. Delete means delete (original+snapshots+embeddings+index+queue+tombstone). Export always works.
9. AI failure never makes archive unavailable. FTS works without embeddings/cloud.
10. No fabricated analytics. Evidence-first. Inferences labeled. Observable facts, not beliefs.
11. The server fetches only public http(s): no private IPs, no metadata
    address, short timeout, small body, few redirects. Fetched pages are
    parsed as text, never executed. Capture answers before any fetch.

# Jev decision-layer correction (Sep 2026)
- Jev = choice/score/boolean with probabilities. NO string generation.
- LLM/code extracts open slots (entities, subjects). Jev judges closed sets only:
  query-router (7 intents: LOOKUP, SEARCH, FILTER, SYNTHESIS, COMPARE,
  RECALL, REDISCOVER), memory-state (10 states), source-classifier (nouls),
  relationship (establishes/contradicts/unrelated), claim-verifier (supported/contradicted/inferred).
- Store choice+probabilities+confidence+model version in decision_events. Thresholds in code.
- Deterministic fast-path for obvious `postgres`-style queries; Jev only for ambiguous.
- Synthesis = retrieval -> LLM prose from verified claims -> Jev claim-check -> evidence UI.
