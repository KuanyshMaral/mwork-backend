-- 000039_align_profiles_with_profile_domain.up.sql
-- Add missing columns used by profile domain code (safe: only add/default/backfill)
-- Split between model_profiles and employer_profiles.

-- =============================================
-- model_profiles: generic model fields
-- =============================================
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS description TEXT NULL;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS country VARCHAR(100) NULL;

ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS categories JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS skills JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS barter_accepted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS accept_remote_work BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS total_reviews INTEGER NOT NULL DEFAULT 0;

-- Ensure visibility default/backfill (000029 adds column but existing rows can still be NULL)
ALTER TABLE model_profiles ALTER COLUMN visibility SET DEFAULT 'public';
UPDATE model_profiles SET visibility = 'public' WHERE visibility IS NULL;

-- Ensure languages default/backfill
ALTER TABLE model_profiles ALTER COLUMN languages SET DEFAULT '[]'::jsonb;
UPDATE model_profiles SET languages = '[]'::jsonb WHERE languages IS NULL;

-- Ensure JSON arrays are not NULL
UPDATE model_profiles SET categories = '[]'::jsonb WHERE categories IS NULL;
UPDATE model_profiles SET skills = '[]'::jsonb WHERE skills IS NULL;

-- travel_cities: convert TEXT[] -> JSONB safely if needed
DO $$
DECLARE
    travel_type text;
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'model_profiles'
          AND column_name = 'travel_cities'
    ) THEN
        SELECT pg_catalog.format_type(a.atttypid, a.atttypmod)
        INTO travel_type
        FROM pg_catalog.pg_attribute a
                 JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
                 JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
        WHERE n.nspname = 'public'
          AND c.relname = 'model_profiles'
          AND a.attname = 'travel_cities'
          AND a.attnum > 0
          AND NOT a.attisdropped;

        IF travel_type LIKE '%[]' THEN
            BEGIN
                ALTER TABLE model_profiles ALTER COLUMN travel_cities DROP DEFAULT;
            EXCEPTION WHEN others THEN
                -- ignore if no default
            END;

            ALTER TABLE model_profiles
                ALTER COLUMN travel_cities TYPE JSONB
                    USING COALESCE(to_jsonb(travel_cities), '[]'::jsonb);

            ALTER TABLE model_profiles
                ALTER COLUMN travel_cities SET DEFAULT '[]'::jsonb;

            UPDATE model_profiles
            SET travel_cities = '[]'::jsonb
            WHERE travel_cities IS NULL;
        ELSE
            BEGIN
                ALTER TABLE model_profiles ALTER COLUMN travel_cities SET DEFAULT '[]'::jsonb;
            EXCEPTION WHEN others THEN
                -- ignore if incompatible
            END;

            BEGIN
                UPDATE model_profiles
                SET travel_cities = '[]'::jsonb
                WHERE travel_cities IS NULL;
            EXCEPTION WHEN others THEN
                -- ignore if incompatible
            END;
        END IF;
    ELSE
        ALTER TABLE model_profiles
            ADD COLUMN travel_cities JSONB NOT NULL DEFAULT '[]'::jsonb;
    END IF;
END $$;

-- =============================================
-- employer_profiles: employer-specific fields
-- =============================================
ALTER TABLE employer_profiles ADD COLUMN IF NOT EXISTS company_type VARCHAR(100) NULL;
ALTER TABLE employer_profiles ADD COLUMN IF NOT EXISTS contact_person VARCHAR(200) NULL;
ALTER TABLE employer_profiles ADD COLUMN IF NOT EXISTS contact_phone VARCHAR(20) NULL;
ALTER TABLE employer_profiles ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE employer_profiles ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ NULL;
