-- 012: search pages wrongly shared one canonical address (params carry
-- their identity). Recompute with tracking-only stripping. Rerunnable.
UPDATE items
SET canonical_url = LOWER(
  CASE WHEN position('?' in raw_ref) = 0 THEN TRIM(TRAILING '/' FROM raw_ref)
  ELSE SPLIT_PART(raw_ref, '?', 1) || '?' || (
    SELECT STRING_AGG(kv, '&') FROM (
      SELECT UNNEST(STRING_TO_ARRAY(SPLIT_PART(raw_ref, '?', 2), '&')) AS kv
    ) s WHERE SPLIT_PART(kv, '=', 1) NOT LIKE 'utm\_%'
      AND SPLIT_PART(kv, '=', 1) NOT IN ('fbclid','gclid','mc_cid')
  ) END)
WHERE source_type IN ('url','bookmark')
  AND (canonical_url LIKE '%/search' OR raw_ref LIKE '%/search?%');
