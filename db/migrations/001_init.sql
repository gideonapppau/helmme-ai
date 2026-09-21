-- Corsa baseline §48: minimal Phase-0 slice.
-- Full domain tables arrive progressively; this supports
-- Milestones 1-6: capture -> canonical -> deterministic -> search.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- NOTE: vector extension + embedding column live in 002_vector.sql so the
-- Phase-0 loop runs on vanilla postgres. pgvector image pull is in progress.

CREATE TABLE IF NOT EXISTS items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT NOT NULL DEFAULT 'local',
  -- Tier 0 capture (§16)
  source_type TEXT NOT NULL, -- url | text | file | pdf | image | bookmark | note
  raw_ref TEXT NOT NULL,     -- original URL, text pointer, or object key
  content_hash TEXT,
  -- Immutable original (§7): never update after insert
  original JSONB NOT NULL,
  -- Tier 1 deterministic (§16)
  canonical_url TEXT,
  title TEXT,
  domain TEXT,
  mime TEXT,
  lang TEXT,
  captured_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  -- Sensitivity (§22)
  sensitivity TEXT NOT NULL DEFAULT 'PERSONAL'
    CHECK (sensitivity IN ('PUBLIC','LOW','PERSONAL','SENSITIVE','HIGHLY_SENSITIVE')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS item_contents (
  item_id UUID PRIMARY KEY REFERENCES items(id) ON DELETE CASCADE,
  extracted_text TEXT,              -- Tier 1 deterministic extraction
  search_tsv TSVECTOR,              -- FTS (§89: postgres FTS first)
  enrichment_tier INT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_contents_tsv ON item_contents USING GIN (search_tsv);
-- Vector index created once real embeddings flow; IVFFlat placeholder:
-- CREATE INDEX ON item_contents USING ivfflat (embedding vector_cosine_ops);

CREATE TABLE IF NOT EXISTS decision_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  item_id UUID REFERENCES items(id) ON DELETE SET NULL,
  component TEXT NOT NULL, -- query-router | source-classifier | claim-verifier ...
  model TEXT NOT NULL,     -- e.g. jev-1.13.0
  input JSONB NOT NULL,
  result JSONB NOT NULL,   -- {choice, probabilities, confidence, ...}
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS provenance_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  item_id UUID REFERENCES items(id) ON DELETE CASCADE,
  kind TEXT NOT NULL, -- captured | canonicalized | enriched | inferred | corrected | deleted
  payload JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
