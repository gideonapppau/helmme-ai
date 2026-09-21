# Docs, everything needed to build this

Source of truth first, interpretation second.

| Doc | What it is |
|---|---|
| `APP_CONSTITUTION.md` | **The constitution (v2.0).** Replaces all prior specs. Read this before anything else. § numbers refer here. |
| `APP_FLOWS.md` | **Movement spec.** Exact data flow, per-platform user journeys, UX contract, skill registry. Build screens against this. |
| `BUILD_PLAN.md` | **Ordered task plan.** Hierarchical, one box at a time. Status: `[x]` done · `[~]` stub · `[ ]` todo. |
| `roadmap.md` | Build phases mapped to repo status. What is built, what is next. |
| `env.md` | Every env key, where to get it, which side (server/browser) may hold it. |
| `eval.md` | Golden dataset plan, metrics, and release gates. |
| `competitors/cleeve.md` | Bookmark-manager teardown. What to steal, what to avoid. |

Conventions:

- Constitution § numbers refer to `APP_CONSTITUTION.md`.
- App copy stays in short, plain sentences. Docs can use full technical language.
- Memory answers "What do I know?"; Jev answers "Given what I know, what should
  I do next?" (Appendix J). Separate brands, shared infra.
- Jev returns choices, scores, and yes/no odds only. Open slots
  (entities, subjects, prose) come from code or a text model, never Jev.
- Thresholds live in code, never in the model. Every Jev call is logged
  to `decision_events` with model version, odds, and confidence.
- The product rule (§91) gates every feature: never make the user work harder
  than the information is worth. Save → machine works.
