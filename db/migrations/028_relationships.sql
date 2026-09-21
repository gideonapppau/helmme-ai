-- 028: stored relationship edges (§32, §72).
-- The related endpoint already computes links on the fly; this table files
-- them with kind, confidence, evidence, and stamps, so every "why this?"
-- can point at a stored row. Same-topic edges join the moment topics exist.
-- Deleting an item wipes its edges with it.
CREATE TABLE IF NOT EXISTS relationships (
  source_item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  target_item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  type TEXT NOT NULL
    CHECK (type IN ('same-site','references','same-topic','saved-together')),
  confidence DOUBLE PRECISION NOT NULL DEFAULT 0.5,
  evidence JSONB NOT NULL DEFAULT '[]',
  model_version TEXT NOT NULL DEFAULT 'deterministic-v1',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  verified_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (source_item_id, target_item_id, type),
  CHECK (source_item_id <> target_item_id)
);
CREATE INDEX IF NOT EXISTS idx_relationships_target ON relationships (target_item_id);
