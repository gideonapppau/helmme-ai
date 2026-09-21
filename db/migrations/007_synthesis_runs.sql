-- 007: synthesis runs (§66). Every generated answer is auditable:
-- what was asked, what evidence went in, which model and prompt version
-- produced it, what came out, and whether its citations checked out.
CREATE TABLE IF NOT EXISTS synthesis_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  query TEXT NOT NULL,
  source_ids JSONB NOT NULL DEFAULT '[]',
  model TEXT NOT NULL DEFAULT '',
  prompt_version TEXT NOT NULL DEFAULT '',
  result TEXT NOT NULL DEFAULT '',
  citations_ok BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_synth_user ON synthesis_runs (user_id, created_at DESC);
