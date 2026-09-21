-- 006: non-URL items must not carry URL metadata. Notes captured before
-- the fix stored their own text as domain; clear it. Rerunnable.
UPDATE items
SET canonical_url = NULL, domain = NULL
WHERE source_type NOT IN ('url','bookmark')
  AND (canonical_url IS NOT NULL OR domain IS NOT NULL);
