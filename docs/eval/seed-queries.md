# Eval seed — 20 queries with known answers (§142)

Run: `powershell -ExecutionPolicy Bypass -File docs/eval/run-eval.ps1`
(add `-Api http://localhost:8081` to point elsewhere).
Pass = expected title in top 3 hits. Run before every search change.
If the archive changes, update expectations — never weaken them silently.

| # | Query | Must find |
|---|---|---|
| 1 | Flutterwave | Gideon_Appau_CV.pdf |
| 2 | anything about marketmate | Gideon_Appau_CV.pdf |
| 3 | Takoradi | Gideon_Appau_CV.pdf |
| 4 | Lighthouse | Gideon_Appau_CV.pdf |
| 5 | Stripe | Gideon_Appau_CV.pdf |
| 6 | Angular | Gideon_Appau_CV.pdf |
| 7 | Clink | Gideon_Appau_CV.pdf |
| 8 | Expo | Gideon_Appau_CV.pdf |
| 9 | ALX | Gideon_Appau_CV.pdf |
| 10 | iCODE | Gideon_Appau_CV.pdf |
| 11 | Maestro | Gideon_Appau_CV.pdf |
| 12 | pooling helmme | pool-test.pdf |
| 13 | quick-save-test | https://example.com/quick-save-test |
| 14 | pgvector guide | Pgvector guide |
| 15 | Go routers | Go routers |
| 16 | remembers why things matter | helmme remembers why things matter |
| 17 | what did I learn about postgres | pool-test.pdf |
| 18 | tell me about payment work | Gideon_Appau_CV.pdf |
| 19 | databases | Gideon_Appau_CV.pdf |
| 20 | offline mobile | Gideon_Appau_CV.pdf |
