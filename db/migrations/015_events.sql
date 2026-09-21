-- 015: open events (§38-39 future "revisited" filter).
-- Logged from today; the filter activates once history exists.
CREATE TABLE IF NOT EXISTS item_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('opened')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_events_item ON item_events (item_id, created_at DESC);
