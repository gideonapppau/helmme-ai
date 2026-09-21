-- 003: uploaded file assets (§60 raw_storage_reference, §13 raw layer).
-- One row per captured file. Bytes live in UPLOAD_DIR (content-addressed);
-- later this becomes an S3-compatible object key without schema change.
CREATE TABLE IF NOT EXISTS item_assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK (kind IN ('pdf','image')),
  storage_ref TEXT NOT NULL,
  mime TEXT NOT NULL,
  byte_size BIGINT NOT NULL,
  width INT,
  height INT,
  page_count INT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_assets_item ON item_assets (item_id);
