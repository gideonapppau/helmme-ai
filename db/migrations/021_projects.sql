-- 021: projects as living views (§33).
-- A project is a named question over the archive, not a folder: confirming a
-- suggestion never moves a single item. Removing a project only removes the
-- view. Suggested rows come from real clusters (topics with 3+ saves);
-- confirmed rows are user-approved or user-declared.
CREATE TABLE IF NOT EXISTS projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'suggested'
    CHECK (status IN ('suggested','confirmed')),
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, name)
);
CREATE INDEX IF NOT EXISTS idx_projects_user ON projects (user_id, status);
