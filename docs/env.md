# Env keys

Copy the `.env.example` next to each app; never commit real values.
`.gitignore` already excludes `.env*`.

## Web (`apps/web/.env.local`)

| Key | Where to get it | Notes |
|---|---|---|
| `TYPESAFE_AI_API_KEY` | TypeSafe console (off waitlist) | Primary. Server-only, no `NEXT_PUBLIC` prefix. |
| `TYPESAFE_API_KEY` | Same | Alternate name, accepted as fallback. |
| `AI_GATEWAY_API_KEY` | vercel.com → AI Gateway → keys | Alternate path (`typesafe-ai/jev`). Not currently wired. |
| `NEXT_PUBLIC_API_URL` |, | Browser → Go API. Default `http://localhost:8081`. Public by design. |

Rule (§92, §49): provider keys are server-only, encrypted at rest when
stored, never logged, never sent to analytics. BYOK keys get the same
treatment when Phase 8 lands.

## API (`services/api/.env`, local runs only) + compose

| Key | Value / source |
|---|---|
| `DATABASE_URL` | compose sets it; local: `postgres://helmme:helmme@localhost:5433/helmme?sslmode=disable` |
| `REDIS_URL` | `redis://localhost:6379` |
| `PORT` | `8081` local / `8080` in compose |
| `UPLOAD_DIR` | compose: `/data/uploads` volume; local default `./data/uploads` |
| `GROQ_API_KEY` | Server-only. Enables synthesis prose. Missing = evidence + plain note, never a failure. Get it at console.groq.com. |
| `SYNTH_MODEL` | Optional, default `openai/gpt-oss-20b`. Check available models if calls fail, names change. |
| `EMBED_KEY` | Server-only. Enables meaning-vectors. Missing = keywords stand alone, never a failure. OpenAI key or compatible host. |
| `EMBED_BASE_URL` | Optional, default `https://api.openai.com/v1`. |
| `EMBED_MODEL` | Optional, default `text-embedding-3-small` (1536 dims, matches migration 002). |
| `X_CLIENT_ID` | X developer portal → app → OAuth 2.0 Client ID. Missing = X routes answer 503 `x_not_configured`, never fake success. Server-only. |
| `X_CLIENT_SECRET` | Same → Client Secret. Server-only, never logged, never returned. |
| `X_REDIRECT_URL` | Same → callback URL, e.g. `http://localhost:8081/v1/sources/x/oauth/callback`. Must match the portal entry exactly. |

Compose sets these itself; the file is only for `go run` outside Docker.
`dev.cmd` exports them when it launches the native binary.

## Ports (all chosen to avoid sibling projects)

| Service | Port | Conflict avoided |
|---|---|---|
| Postgres | 5433 | whotops-db on 5432 |
| API | 8081 | clerk-core on 8080 |
| Web | 3001 | clerk-gateway on 3000 |
| Redis | 6379 |, |
