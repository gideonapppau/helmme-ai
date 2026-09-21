-- 022: per-user behavior settings (§38).
-- Resurfacing mood lives on the server so every client behaves the same:
-- Quiet shows almost nothing, Balanced some, Proactive more. Default Quiet.
-- One row per user, created on first save.
CREATE TABLE IF NOT EXISTS user_settings (
  user_id TEXT PRIMARY KEY,
  resurface_mode TEXT NOT NULL DEFAULT 'quiet'
    CHECK (resurface_mode IN ('quiet','balanced','proactive')),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
