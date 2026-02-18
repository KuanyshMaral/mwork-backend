-- 000029_add_profile_visibility.up.sql

-- Add visibility column to model_profiles
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS visibility VARCHAR(20) DEFAULT 'public';

-- Add check constraint for allowed visibility values
ALTER TABLE model_profiles ADD CONSTRAINT model_profiles_visibility_check
    CHECK (visibility IN ('public', 'link_only', 'hidden'));

-- Add comment to column
COMMENT ON COLUMN model_profiles.visibility IS 'Profile visibility: public, link_only, or hidden';
