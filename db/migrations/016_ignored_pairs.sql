-- 016: dismissed duplicate pairs (§35). Ignored pairs never resurface.
-- No un-ring in v1: dismissed stays dismissed until a surface exists
-- to manage it.
CREATE TABLE IF NOT EXISTS ignored_pairs (
  user_id TEXT NOT NULL DEFAULT 'local',
  a_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  b_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (a_id < b_id),
  UNIQUE (user_id, a_id, b_id)
);
CREATE INDEX IF NOT EXISTS idx_ignored_user ON ignored_pairs (user_id);
