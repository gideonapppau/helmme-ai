# helmme-ai — personal context prototype

> Save anything. Find what you meant later.

Constitution: `docs/APP_CONSTITUTION.md` (v2.0 — replaces all prior specs).

## Stack (constitution §67–§70)

Next.js+TS web · Go API · Postgres+pgvector+FTS · Redis · S3-compatible storage ·
Jev decision layer as separate brand (Appendix J: judges closed sets only, via `experimental_evaluate`).

## Run Phase-0 slice

```powershell
docker compose up --build
# apply SQL in order:
Get-ChildItem db/migrations/*.sql | Sort-Object Name | ForEach-Object { Get-Content $_.FullName | docker exec -i helmme-ai-postgres-1 psql -U helmme -d helmme }
cd apps/web; npm install; npm run dev
```

API: `POST /v1/items`, `POST /v1/search`, `GET /v1/recent`, `GET /v1/archive`,
`GET /v1/sources`, `POST /v1/sources/x/bookmarks/import`.
Web: Search · Recent · Capture · Archive · Settings (§3). Synthesis is explicit escalation.

## Invariants (constitution §91 + App. J)

Save → Done. Original immutable, capture≠enrichment, raw wins over derived,
Jev judges closed sets only, memory ("what do I know?") separate from Jev
("what should I do?").
