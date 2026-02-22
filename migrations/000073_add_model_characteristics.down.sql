-- Reverse migration 000073

-- Restore profile_social_links table
CREATE TABLE IF NOT EXISTS profile_social_links (
    id UUID PRIMARY KEY,
    profile_id UUID NOT NULL REFERENCES model_profiles(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    url TEXT NOT NULL,
    username VARCHAR(100),
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_social_profile ON profile_social_links(profile_id);

-- Remove casting columns
ALTER TABLE castings
    DROP COLUMN IF EXISTS required_hair_colors,
    DROP COLUMN IF EXISTS required_eye_colors;

-- Remove employer_profiles columns
ALTER TABLE employer_profiles
    DROP COLUMN IF EXISTS social_links,
    DROP COLUMN IF EXISTS profile_views;

-- Remove model_profiles columns
ALTER TABLE model_profiles
    DROP COLUMN IF EXISTS hair_color,
    DROP COLUMN IF EXISTS eye_color,
    DROP COLUMN IF EXISTS tattoos,
    DROP COLUMN IF EXISTS working_hours,
    DROP COLUMN IF EXISTS min_budget,
    DROP COLUMN IF EXISTS social_links;
