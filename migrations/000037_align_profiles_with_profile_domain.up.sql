-- 000037_align_profiles_with_profile_domain.up.sql
-- Add missing columns used by profile domain code (safe: only add/default/backfill)

ALTER TABLE profiles ADD COLUMN IF NOT EXISTS description TEXT NULL;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS country VARCHAR(100) NULL;

ALTER TABLE profiles ADD COLUMN IF NOT EXISTS categories JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS skills JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE profiles ADD COLUMN IF NOT EXISTS barter_accepted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS accept_remote_work BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE profiles ADD COLUMN IF NOT EXISTS total_reviews INTEGER NOT NULL DEFAULT 0;

-- Employer fields used by code
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS company_type VARCHAR(100) NULL;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS contact_person VARCHAR(200) NULL;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(20) NULL;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ NULL;

-- Ensure visibility default/backfill (000029 adds column but existing rows can still be NULL)
ALTER TABLE profiles ALTER COLUMN visibility SET DEFAULT 'public';
UPDATE profiles SET visibility = 'public' WHERE visibility IS NULL;

-- Ensure languages default/backfill
ALTER TABLE profiles ALTER COLUMN languages SET DEFAULT '[]'::jsonb;
UPDATE profiles SET languages = '[]'::jsonb WHERE languages IS NULL;

-- Ensure JSON arrays are not NULL (in case column existed but had NULLs)
UPDATE profiles SET categories = '[]'::jsonb WHERE categories IS NULL;
UPDATE profiles SET skills = '[]'::jsonb WHERE skills IS NULL;

-- travel_cities:
-- repo expects JSON-like arrays per Stage2; but earlier migration may have created TEXT[].
-- Convert TEXT[] -> JSONB safely; otherwise ensure JSONB default/backfill.
DO $$
DECLARE
travel_type text;
BEGIN
    -- column exists?
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'profiles'
          AND column_name = 'travel_cities'
    ) THEN

        -- detect actual type (works for arrays/jsonb/etc.)
SELECT pg_catalog.format_type(a.atttypid, a.atttypmod)
INTO travel_type
FROM pg_catalog.pg_attribute a
         JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
         JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public'
  AND c.relname = 'profiles'
  AND a.attname = 'travel_cities'
  AND a.attnum > 0
  AND NOT a.attisdropped;

-- If it is text[] (or any array), convert to jsonb
IF travel_type LIKE '%[]' THEN
            -- IMPORTANT: drop old default (e.g. '{}'::text[]) before type change
BEGIN
ALTER TABLE profiles ALTER COLUMN travel_cities DROP DEFAULT;
EXCEPTION WHEN others THEN
                -- ignore if no default
END;

ALTER TABLE profiles
ALTER COLUMN travel_cities TYPE JSONB
                USING COALESCE(to_jsonb(travel_cities), '[]'::jsonb);

ALTER TABLE profiles
    ALTER COLUMN travel_cities SET DEFAULT '[]'::jsonb;

UPDATE profiles
SET travel_cities = '[]'::jsonb
WHERE travel_cities IS NULL;

ELSE
            -- If already JSONB (or other), just try to set default/backfill
BEGIN
ALTER TABLE profiles ALTER COLUMN travel_cities SET DEFAULT '[]'::jsonb;
EXCEPTION WHEN others THEN
                -- ignore if incompatible
END;

BEGIN
UPDATE profiles
SET travel_cities = '[]'::jsonb
WHERE travel_cities IS NULL;
EXCEPTION WHEN others THEN
                -- ignore if incompatible
END;
END IF;

ELSE
        -- Column doesn't exist: create it as JSONB
ALTER TABLE profiles
    ADD COLUMN travel_cities JSONB NOT NULL DEFAULT '[]'::jsonb;
END IF;
END $$;
