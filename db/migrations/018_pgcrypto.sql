-- 018: pgcrypto for gen_random_uuid() (used as UUID default since 001).
-- 001 only ensured pg_trgm; fresh databases on vanilla postgres fail the
-- first insert without this. Idempotent: safe on already-migrated DBs.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
