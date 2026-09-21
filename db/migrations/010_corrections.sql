-- 010: user corrections (§31, §157). Rejections are remembered so
-- enrichment never resurrects what the user removed.
CREATE TABLE IF NOT EXISTS user_corrections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  item_id UUID REFERENCES items(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('topic','entity')),
  target_id UUID NOT NULL,
  action TEXT NOT NULL DEFAULT 'reject',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, item_id, kind, target_id)
);
CREATE INDEX IF NOT EXISTS idx_corrections_item ON user_corrections (item_id);
