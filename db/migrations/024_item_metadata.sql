-- 024: derived facts live apart from raw captures (§42).
-- The `original` column on items is the user's bytes and stays byte-identical
-- to what arrived. Everything the machine guesses (fetched author, fetched
-- publish date, later: model labels) goes here instead, each with where it
-- came from and how sure it is. Deleting the item wipes its guesses too.
-- One-time move: existing fetched_* keys are lifted out of `original`.
CREATE TABLE IF NOT EXISTS item_metadata (
  item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  value TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  model_version TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (item_id, key)
);
INSERT INTO item_metadata (item_id, key, value, source, confidence)
SELECT id, 'fetched_author', original->>'fetched_author', 'fetch', 0.7
FROM items WHERE original ? 'fetched_author'
ON CONFLICT (item_id, key) DO NOTHING;
INSERT INTO item_metadata (item_id, key, value, source, confidence)
SELECT id, 'fetched_published', original->>'fetched_published', 'fetch', 0.7
FROM items WHERE original ? 'fetched_published'
ON CONFLICT (item_id, key) DO NOTHING;
UPDATE items SET original = original - 'fetched_author' - 'fetched_published'
WHERE original ? 'fetched_author' OR original ? 'fetched_published';
