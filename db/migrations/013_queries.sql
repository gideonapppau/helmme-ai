-- 013: saved questions (§31 collections-as-queries, §67).
-- A view stores the question plus what it last showed, so "+N new"
-- is a subtraction, not a feature.
CREATE TABLE IF NOT EXISTS queries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  name TEXT NOT NULL,
  query_text TEXT NOT NULL,
  last_run_at TIMESTAMPTZ,
  last_count INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_queries_user ON queries (user_id, created_at DESC);
