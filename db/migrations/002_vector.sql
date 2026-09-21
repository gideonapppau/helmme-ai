-- 002: Tier-2 semantic layer. Apply once running pgvector/pgvector:pg16.
-- Kept separate so Phase-0 capture->FTS works on vanilla postgres.
CREATE EXTENSION IF NOT EXISTS vector;
ALTER TABLE item_contents ADD COLUMN IF NOT EXISTS embedding vector(1536);
-- CREATE INDEX ON item_contents USING ivfflat (embedding vector_cosine_ops);
