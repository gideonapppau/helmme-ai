# Eval, golden dataset and release gates (§142-144)

No ranking or synthesis change ships without running the dataset.

## Golden archive (private, hand-built)

- notes, articles, PDFs, screenshots
- exact + near + conceptual duplicates
- contradictory notes on one topic (dated)
- temporal drift (position changes over months)
- unrelated distractors sharing keywords

## Question set (§142 examples)

Where / what did I save about X? What did I say about X? What did I
learn about X? What supports X? What changed in my thinking on X?
Which notes relate to X?

## Metrics

- Retrieval: recall@K, precision@K, MRR, per query, tracked over time.
- Synthesis: citation correctness (% claims with supporting cited
  source), groundedness (% claims in Evidence vs Inference vs Unknown),
  hallucination rate (unsupported personal assertions = 0 tolerance).
- Extraction: entity/topic precision on labeled items; duplicate
  precision/recall; temporary-vs-durable accuracy.
- Product (§119): capture success, retrieval success, search-to-source
  opens, citation inspections, 7/30/90-day retention.

## Gates (§144), block release on

- retrieval regression vs baseline
- any citation hallucination in the dataset
- silent data loss, duplicate corruption, sync corruption
- privacy boundary violation (untrusted content reaching personal
  context, key material in logs, cross-tenant read)

## Harness

- `services/api` unit tests: parsing, canonicalization, validation,
  authz (`go test ./...`).
- `packages/context-engine` policy tests + Jev mocks (`node --test`),
  thresholds asserted, never-guess behavior asserted.
- Live Jev spot-checks on a fixed 20-query slice before prompt/criteria
  edits; criteria edits are code changes, reviewed as such.
