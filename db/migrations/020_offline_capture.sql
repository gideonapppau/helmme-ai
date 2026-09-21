-- 020: offline-proof capture (§39, Phase 2).
-- A phone with no signal saves with its own local id (client_id) and the real
-- moment of capture (captured_at). When it retries later, the server spots the
-- same client_id and returns the first save instead of making a twin.
-- The partial index only touches rows that carry a client_id, so old rows
-- (client_id '') are left completely alone.
ALTER TABLE items ADD COLUMN IF NOT EXISTS client_id TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_items_client
  ON items (user_id, client_id) WHERE client_id <> '';
