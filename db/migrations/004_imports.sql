-- 004: bookmark/file import runs (§53, §97 import experience).
-- Every run records honest numbers: imported vs already-saved vs skipped.
CREATE TABLE IF NOT EXISTS imports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  source TEXT NOT NULL,             -- browser-bookmarks | file
  filename TEXT NOT NULL,
  total INT NOT NULL DEFAULT 0,
  imported INT NOT NULL DEFAULT 0,
  duplicates INT NOT NULL DEFAULT 0,
  skipped INT NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'done',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS import_items (
  import_id UUID NOT NULL REFERENCES imports(id) ON DELETE CASCADE,
  item_id UUID REFERENCES items(id) ON DELETE SET NULL,
  canonical_url TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('imported','duplicate','skipped')),
  note TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_import_items_run ON import_items (import_id);
