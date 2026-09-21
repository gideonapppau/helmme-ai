# Docs — everything needed to build this

Source of truth first, interpretation second.

| Doc | What it is |
|---|---|
| `spec/product-spec-v1.0.md` | The locked product spec, verbatim. Read this before anything else. |
| `../ARCHITECTURE.md` | 25 engineering invariants + the Jev correction (Jev judges, never generates). Non-negotiable. |
| `jev-decision-layer.md` | Where Jev plugs into the spec's pipeline, exact question sets, thresholds, cost policy, and what Jev must never do. |
| `roadmap.md` | V0 phases mapped to repo status. What is built, what is next. |
| `env.md` | Every env key, where to get it, which side (server/browser) may hold it. |
| `eval.md` | Golden dataset plan, metrics, and release gates. |
| `competitors/cleeve.md` | Bookmark-manager teardown. What to steal, what to avoid. |

Conventions:

- Spec section numbers (§) refer to `spec/product-spec-v1.0.md`.
- Invariant numbers refer to `ARCHITECTURE.md`.
- App copy stays in short, plain sentences. Docs can use full technical language.
- Jev returns choices, scores, and yes/no odds only. Open slots
  (entities, subjects, prose) come from code or a text model, never Jev.
- Thresholds live in code, never in the model. Every Jev call is logged
  to `decision_events` with model version, odds, and confidence.
