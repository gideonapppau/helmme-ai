-- 025: topic controls (§34).
-- Topics are machine guesses, so the user can overrule them: rename a bad
-- name, hide a noisy one, merge two that mean the same thing. Hidden topics
-- stop surfacing everywhere (search chips, related, resurface, suggestions)
-- but their items stay exactly where they are.
ALTER TABLE topics ADD COLUMN IF NOT EXISTS hidden BOOLEAN NOT NULL DEFAULT FALSE;
