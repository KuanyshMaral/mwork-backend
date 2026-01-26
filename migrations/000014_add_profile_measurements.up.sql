-- Migration: Add profile measurements and social links
-- Purpose: Support detailed model parameters display and social media links

-- Physical measurements (in cm)
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS bust_cm INT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS waist_cm INT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS hips_cm INT;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS shoe_size DECIMAL(3,1);

-- Appearance
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS hair_color VARCHAR(50);
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS eye_color VARCHAR(50);
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS skin_tone VARCHAR(50);

-- Specializations array
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS specializations TEXT[] DEFAULT '{}';

-- Denormalized stats for fast dashboard display
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS total_earnings BIGINT DEFAULT 0;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS completed_jobs INT DEFAULT 0;
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS profile_views INT DEFAULT 0;

-- Indexes for search/filter
CREATE INDEX IF NOT EXISTS idx_profiles_specializations ON profiles USING GIN(specializations);
CREATE INDEX IF NOT EXISTS idx_profiles_measurements ON profiles(bust_cm, waist_cm, hips_cm);
CREATE INDEX IF NOT EXISTS idx_profiles_hair ON profiles(hair_color) WHERE hair_color IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_profiles_eye ON profiles(eye_color) WHERE eye_color IS NOT NULL;

-- Social links table
CREATE TABLE IF NOT EXISTS profile_social_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    url TEXT NOT NULL,
    username VARCHAR(100),
    is_verified BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_platform CHECK (platform IN (
        'instagram', 'tiktok', 'facebook', 'twitter', 
        'youtube', 'telegram', 'linkedin', 'vk'
    )),
    UNIQUE(profile_id, platform)
);

CREATE INDEX IF NOT EXISTS idx_social_profile ON profile_social_links(profile_id);

-- Comments
COMMENT ON COLUMN profiles.specializations IS 'Array of specialization tags: fashion, commercial, beauty, etc.';
COMMENT ON COLUMN profiles.total_earnings IS 'Denormalized total earnings for fast display';
COMMENT ON COLUMN profiles.completed_jobs IS 'Denormalized completed jobs count';
COMMENT ON TABLE profile_social_links IS 'Social media links for model profiles';
