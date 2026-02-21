-- Migration 000073: Add model characteristics and employer profile enhancements
-- 1. Add missing physical/professional attributes to model_profiles
-- 2. Add social_links and profile_views to employer_profiles
-- 3. Drop the redundant profile_social_links table (replaced by JSONB column)
-- 4. Add hair/eye color requirement filters to castings

-- ============================================================
-- model_profiles: new characteristics
-- ============================================================
ALTER TABLE model_profiles
    ADD COLUMN IF NOT EXISTS hair_color     VARCHAR(50),
    ADD COLUMN IF NOT EXISTS eye_color      VARCHAR(50),
    ADD COLUMN IF NOT EXISTS tattoos        VARCHAR(255),
    ADD COLUMN IF NOT EXISTS working_hours  VARCHAR(150),
    ADD COLUMN IF NOT EXISTS min_budget     DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS social_links   JSONB NOT NULL DEFAULT '[]'::jsonb;

-- ============================================================
-- employer_profiles: social links + profile views
-- ============================================================
ALTER TABLE employer_profiles
    ADD COLUMN IF NOT EXISTS social_links   JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS profile_views  INTEGER NOT NULL DEFAULT 0;

-- ============================================================
-- castings: new requirement filters for hair and eye color
-- ============================================================
ALTER TABLE castings
    ADD COLUMN IF NOT EXISTS required_hair_colors TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS required_eye_colors  TEXT[] NOT NULL DEFAULT '{}';

-- ============================================================
-- Drop the now-redundant profile_social_links table
-- This functionality is replaced by the social_links JSONB column
-- on model_profiles and employer_profiles.
-- ============================================================
DROP TABLE IF EXISTS profile_social_links;
