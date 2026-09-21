-- 017: constitution v2.0 alignment (§11-§14 X-first, §39-§40 sync, §49 SourceConnector, §53 disconnect).
-- Sources registry + per-item sync state. Additive only; existing rows default sensibly.
CREATE TABLE IF NOT EXISTS sources (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  kind TEXT NOT NULL,              -- x | browser | apple_notes | reddit | youtube | file | share_sheet | manual
  status TEXT NOT NULL DEFAULT 'disconnected'
    CHECK (status IN ('disconnected','connecting','connected','syncing','error')),
  external_ref TEXT NOT NULL DEFAULT '',  -- oauth account id / handle, never a password (§11)
  total INT NOT NULL DEFAULT 0,
  imported INT NOT NULL DEFAULT 0,
  last_sync_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, kind)
);

-- Sync state machine (§40): local · pending · synced · conflict · failed · deleted.
ALTER TABLE items ADD COLUMN IF NOT EXISTS sync_state TEXT NOT NULL DEFAULT 'synced'
  CHECK (sync_state IN ('local','pending','synced','conflict','failed','deleted'));
ALTER TABLE items ADD COLUMN IF NOT EXISTS server_id TEXT NOT NULL DEFAULT '';
ALTER TABLE items ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1;
ALTER TABLE items ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE items ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE items ADD COLUMN IF NOT EXISTS source_id UUID REFERENCES sources(id) ON DELETE SET NULL;

-- Disconnect policy (§53): keep vs delete is a user choice recorded per source.
ALTER TABLE sources ADD COLUMN IF NOT EXISTS disconnected_at TIMESTAMPTZ;
ALTER TABLE sources ADD COLUMN IF NOT EXISTS on_disconnect TEXT NOT NULL DEFAULT 'keep'
  CHECK (on_disconnect IN ('keep','delete'));

CREATE INDEX IF NOT EXISTS idx_items_sync ON items (user_id, sync_state);
CREATE INDEX IF NOT EXISTS idx_items_source ON items (user_id, source_type);
CREATE INDEX IF NOT EXISTS idx_sources_user ON sources (user_id);
