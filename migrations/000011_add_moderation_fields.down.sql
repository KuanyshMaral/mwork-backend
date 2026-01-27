-- 000011_add_moderation_fields.down.sql

-- Users
ALTER TABLE users DROP COLUMN IF EXISTS is_banned;
ALTER TABLE users DROP COLUMN IF EXISTS banned_at;
ALTER TABLE users DROP COLUMN IF EXISTS banned_reason;
ALTER TABLE users DROP COLUMN IF EXISTS banned_by;
ALTER TABLE users DROP COLUMN IF EXISTS is_verified;
ALTER TABLE users DROP COLUMN IF EXISTS is_identity_verified;
ALTER TABLE users DROP COLUMN IF EXISTS verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS verified_by;

-- Profiles
ALTER TABLE profiles DROP COLUMN IF EXISTS moderation_status;
ALTER TABLE profiles DROP COLUMN IF EXISTS moderated_at;
ALTER TABLE profiles DROP COLUMN IF EXISTS moderated_by;
ALTER TABLE profiles DROP COLUMN IF EXISTS moderation_note;

-- Castings
ALTER TABLE castings DROP COLUMN IF EXISTS moderation_status;
ALTER TABLE castings DROP COLUMN IF EXISTS moderated_at;
ALTER TABLE castings DROP COLUMN IF EXISTS moderated_by;
ALTER TABLE castings DROP COLUMN IF EXISTS moderation_note;
ALTER TABLE castings DROP COLUMN IF EXISTS is_featured;

-- Photos
ALTER TABLE photos DROP COLUMN IF EXISTS moderation_status;
ALTER TABLE photos DROP COLUMN IF EXISTS moderated_at;
ALTER TABLE photos DROP COLUMN IF EXISTS moderated_by;
ALTER TABLE photos DROP COLUMN IF EXISTS is_nsfw;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_banned;
DROP INDEX IF EXISTS idx_profiles_moderation;
DROP INDEX IF EXISTS idx_castings_moderation;
DROP INDEX IF EXISTS idx_castings_featured;
DROP INDEX IF EXISTS idx_photos_moderation;
