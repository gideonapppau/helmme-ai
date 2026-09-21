# helmme-ai — Corsa personal-memory prototype

> Save anything. Organize nothing. Find everything.

## Stack (locked §86-91)
Next.js+TS web · Go API · Postgres+pgvector+FTS · Redis · S3-compatible storage · Jev decision layer via `experimental_evaluate`.

## Run Phase-0 slice
```powershell
docker compose up --build
# apply SQL:
Get-Content db/migrations/001_init.sql | docker exec -i helmme-ai-postgres-1 psql -U helmme -d helmme
cd apps/web; npm install; npm run dev
```
API: `POST /v1/items`, `POST /v1/search`. Web omnibar: search-first, synthesis explicit escalation.

## Invariants
See `docs/ARCHITECTURE.md`. Original immutable, capture≠enrichment, Jev judges closed sets only.
