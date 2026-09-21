-- 009: context graph v1 (§§62-64). Topics and entities with confidence
-- and source; links are idempotent (re-enrich never duplicates).
-- Entity-to-entity relationships arrive with graph density, not here.
CREATE TABLE IF NOT EXISTS topics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  name TEXT NOT NULL,
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  source TEXT NOT NULL DEFAULT 'deterministic',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, name)
);
CREATE TABLE IF NOT EXISTS entities (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  type TEXT NOT NULL,
  canonical_name TEXT NOT NULL,
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  source TEXT NOT NULL DEFAULT 'deterministic',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, type, canonical_name)
);
CREATE TABLE IF NOT EXISTS item_topics (
  item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  PRIMARY KEY (item_id, topic_id)
);
CREATE TABLE IF NOT EXISTS item_entities (
  item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  PRIMARY KEY (item_id, entity_id)
);
CREATE INDEX IF NOT EXISTS idx_item_topics_topic ON item_topics (topic_id);
CREATE INDEX IF NOT EXISTS idx_item_entities_entity ON item_entities (entity_id);
