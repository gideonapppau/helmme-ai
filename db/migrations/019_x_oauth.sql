-- 019: X OAuth token store + backfill progress (§11–§14, §49).
-- Tokens live server-side only: never logged, never returned by any endpoint.
-- At-rest encryption for stored secrets lands with Phase 11 (account work);
-- until then this table is reachable only via the API's server-side queries.
-- All columns additive; safe on already-migrated DBs.
ALTER TABLE sources ADD COLUMN IF NOT EXISTS access_token TEXT NOT NULL DEFAULT '';
ALTER TABLE sources ADD COLUMN IF NOT EXISTS refresh_token TEXT NOT NULL DEFAULT '';
ALTER TABLE sources ADD COLUMN IF NOT EXISTS token_expires_at TIMESTAMPTZ;
ALTER TABLE sources ADD COLUMN IF NOT EXISTS token_scope TEXT NOT NULL DEFAULT '';
ALTER TABLE sources ADD COLUMN IF NOT EXISTS oauth_state TEXT NOT NULL DEFAULT '';
ALTER TABLE sources ADD COLUMN IF NOT EXISTS code_verifier TEXT NOT NULL DEFAULT '';
-- Pollable backfill progress (§12: 643 / 2,184, user free to leave).
ALTER TABLE sources ADD COLUMN IF NOT EXISTS backfill_status TEXT NOT NULL DEFAULT 'idle'
  CHECK (backfill_status IN ('idle','running','done','error'));
ALTER TABLE sources ADD COLUMN IF NOT EXISTS backfill_note TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_sources_state ON sources (oauth_state) WHERE oauth_state <> '';
