-- 011: reindex URL captures without tracking soup. Stored text becomes
-- title + de-slugged canonical URL; raw_ref keeps the full address for
-- opening. Rerunnable.
UPDATE item_contents c
SET extracted_text = i.title || E'\n' || COALESCE(i.canonical_url, ''),
    search_tsv = to_tsvector('english',
      i.title || ' ' || COALESCE(i.canonical_url,'') || ' ' ||
      regexp_replace(COALESCE(i.title,'') || ' ' || COALESCE(i.canonical_url,''),'[^a-zA-Z0-9]+',' ','g'))
FROM items i
WHERE c.item_id = i.id AND i.source_type IN ('url','bookmark');
