-- 014: merge state (§35). Merged items hide everywhere but keep their
-- rows, so unmerge restores byte-identical state. Existing rows are active.
ALTER TABLE items ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE items ADD COLUMN IF NOT EXISTS merged_into UUID REFERENCES items(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_items_status ON items (user_id, status);
