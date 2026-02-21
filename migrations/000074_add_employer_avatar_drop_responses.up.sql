-- Add avatar support for employers
ALTER TABLE employer_profiles
    ADD COLUMN IF NOT EXISTS avatar_upload_id UUID
        REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_employer_profiles_avatar_upload_id
    ON employer_profiles(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;

COMMENT ON COLUMN employer_profiles.avatar_upload_id IS '1:1 FK to uploads. NULL means no avatar set.';

-- Drop legacy responses table
DROP TABLE IF EXISTS responses CASCADE;
