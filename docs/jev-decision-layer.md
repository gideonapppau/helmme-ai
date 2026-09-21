# Jev decision layer — mapping to the v1.0 spec

Jev is the judgment layer of the Context Engine. It makes small, typed
decisions with odds. It never writes prose, extracts names, or answers the
user. Code retrieves, a text model writes, Jev judges.

Live access: direct TypeSafe key (`TYPESAFE_AI_API_KEY`), model
`jev-latest`, via `experimental_evaluate`. See `env.md`.

## Where Jev plugs in (§14 pipeline)

```
Captured Item → Normalize → Extract Metadata        (code, Tier 1)
  → Memory-State Classify                            (Jev choice, 10 states)
  → Entity/Topic extraction                          (text model or code — NOT Jev)
  → Relationship detection                           (Jev choice + confidence)
  → Duplicate detection                              (Jev yes/no + score)
  → Temporary vs durable                             (Jev choice: Temporary / Likely durable / Unknown)
  → Index → Context Graph (odds + evidence stored, §18)
```

## Intent router (§22)

Spec taxonomy is the question set — 7 intents, fixed:

```
LOOKUP | SEARCH | FILTER | SYNTHESIZE | COMPARE | RECALL | REDISCOVER
```

Rules answer obvious queries first (`postgres` → SEARCH, no call).
Jev answers ambiguous ones only. Intent taxonomy is the spec's §22:
LOOKUP, SEARCH, FILTER, SYNTHESIS, COMPARE, RECALL, REDISCOVER —
single-sourced in `@helmme/context-engine`, consumed by the web route
and page. Live-verified: COMPARE at 1.0 confidence, FILTER correctly
held for review at 0.59.

Progressive understanding (§23): the UI offers synthesis as an explicit
escalation. Jev's `needs_synthesis` yes/no decides whether to show it.

## Synthesis gate (§26–29, §73–75)

```
Query → Intent → Candidates → Rerank → Relationship expand
  → Evidence select → Compress → Text model drafts
  → Citation gate: every citation exists (code), every cited source
    supports its claim (Jev), unsupported claims flagged (Jev)
  → Answer with Evidence / Inference / Unknown labels (§28, §75)
```

Hard rule (§29): low support across the board → the system says
"I found N sources, but they aren't enough." Never false confidence.

`SynthesisRun` (§66) stores model + prompt version + citations so any
answer can be audited later.

## Maintenance (§36–37) and contradictions (§87)

Jev classifiers per item: `is_duplicate`, `is_temporary`,
`is_obsolete`, `contradicts(item B)`. All suggestions, never actions —
the user approves every delete or merge (§157 reversibility).
Contradictions surface both sides with dates; the system never picks
a winner.

## Relationships (§18, §34, §64)

Every relationship stores: type, confidence, evidence item ids,
created/verified timestamps. Jev choice among the §34 types, or
`unrelated`. Confidence below bar → tentative or dropped, never
presented as fact.

## Cost policy (§141)

```
deterministic → code (free)
bounded judgment → Jev (~$0.042/1M in, no output charge)
prose/reasoning → text model, synthesis only, evidence-limited context
```

Batch parallel questions in one Jev call (§73 context assembly stays
small — never dump the archive into a model).

## What Jev must never do

- Extract entities, subjects, dates, or any open string.
- Write summaries, answers, or any prose.
- Decide permissions, act on tools, or touch credentials.
- Serve as a security boundary — the app validates everything (§50, §76).
- Be the only decider on destructive or irreversible actions.
