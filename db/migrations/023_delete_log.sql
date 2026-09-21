-- 023: delete tombstones (§52, future sync).
-- A hard delete wipes the item and everything hanging off it (all child rows
-- cascade). This log is written first and answers to nobody: no foreign key,
-- so it survives the delete. Sync (Phase 11) will read it to tell other
-- devices "this is gone on purpose". Users never see it.
CREATE TABLE IF NOT EXISTS delete_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  item_id UUID NOT NULL,
  content_hash TEXT NOT NULL DEFAULT '',
  deleted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_delete_log_user ON delete_log (user_id, deleted_at DESC);
