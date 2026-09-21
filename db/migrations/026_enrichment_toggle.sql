-- 026: enrichment on/off switch (§41 user control).
-- Turning enrichment off never breaks saving or searching: captures still
-- land, index, and stay searchable. Only the machine's guesses (topics,
-- entities, relations) stop. Default on.
ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS ai_enrichment BOOLEAN NOT NULL DEFAULT TRUE;
