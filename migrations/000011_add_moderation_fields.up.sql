-- 000011_add_moderation_fields.up.sql
-- Add moderation and admin fields to existing tables

-- Users: ban and verification fields
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_banned BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS banned_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS banned_reason TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS banned_by UUID;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_identity_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS verified_by UUID;

-- model_profiles: moderation status
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(20) DEFAULT 'pending';
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS moderated_by UUID;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS moderation_note TEXT;

-- Castings: moderation and featuring
ALTER TABLE castings ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(20) DEFAULT 'approved';
ALTER TABLE castings ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS moderated_by UUID;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS moderation_note TEXT;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS is_featured BOOLEAN DEFAULT FALSE;

-- Photos: moderation and NSFW flag
ALTER TABLE photos ADD COLUMN IF NOT EXISTS moderation_status VARCHAR(20) DEFAULT 'pending';
ALTER TABLE photos ADD COLUMN IF NOT EXISTS moderated_at TIMESTAMPTZ;
ALTER TABLE photos ADD COLUMN IF NOT EXISTS moderated_by UUID;
ALTER TABLE photos ADD COLUMN IF NOT EXISTS is_nsfw BOOLEAN DEFAULT FALSE;

-- Indexes for moderation queries
CREATE INDEX IF NOT EXISTS idx_users_banned ON users(is_banned) WHERE is_banned = true;
CREATE INDEX IF NOT EXISTS idx_model_profiles_moderation ON model_profiles(moderation_status);
CREATE INDEX IF NOT EXISTS idx_castings_moderation ON castings(moderation_status);
CREATE INDEX IF NOT EXISTS idx_castings_featured ON castings(is_featured) WHERE is_featured = true;
CREATE INDEX IF NOT EXISTS idx_photos_moderation ON photos(moderation_status);

-- Comments
COMMENT ON COLUMN users.is_banned IS 'Whether user is banned from the platform';
COMMENT ON COLUMN model_profiles.moderation_status IS 'pending, approved, rejected';
COMMENT ON COLUMN castings.is_featured IS 'Featured castings shown prominently';
COMMENT ON COLUMN photos.is_nsfw IS 'Flagged as not safe for work';
