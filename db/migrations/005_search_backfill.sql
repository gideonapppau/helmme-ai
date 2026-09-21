-- 005: de-slugged search vectors for rows indexed before the fix.
-- URL slugs (/quick-save-test) and hyphenated terms become findable.
-- Rerunnable.
UPDATE item_contents
SET search_tsv = to_tsvector(
  'english',
  COALESCE(extracted_text,'') || ' ' ||
  regexp_replace(COALESCE(extracted_text,''),'[^a-zA-Z0-9]+',' ','g')
);
