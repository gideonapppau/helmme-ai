-- 008: per-claim verdicts on synthesis runs (§28 trust layer).
-- Each claim carries its Jev odds so labels stay auditable, not vibes.
ALTER TABLE synthesis_runs ADD COLUMN IF NOT EXISTS claims JSONB NOT NULL DEFAULT '[]';
