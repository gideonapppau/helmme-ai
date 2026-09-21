-- 027: usage metrics (§80 north star).
-- The product wins when finding works, not when collecting grows. So every
-- search leaves a small honest trace: what was asked, how many answers, was
-- it a stretch. Syntheses, opens, and captures already have their own tables
-- (synthesis_runs, item_events, items); this one completes the picture.
-- Internal eyes only: no endpoint exposes other users, and there is one user.
CREATE TABLE IF NOT EXISTS search_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  query TEXT NOT NULL DEFAULT '',
  hits INT NOT NULL DEFAULT 0,
  relaxed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_search_events_user ON search_events (user_id, created_at DESC);
