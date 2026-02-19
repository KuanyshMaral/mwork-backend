-- 000071_add_avatar_and_cover_fks.up.sql
-- Phase 4: Add direct FK columns for 1:1 upload relationships.
-- Rule A: 1:1 → FK column on the owning entity (no intermediate table needed).

-- Model avatar: model_profiles.avatar_upload_id → uploads.id
ALTER TABLE model_profiles
    ADD COLUMN IF NOT EXISTS avatar_upload_id UUID
        REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_model_profiles_avatar_upload_id
    ON model_profiles(avatar_upload_id)
    WHERE avatar_upload_id IS NOT NULL;

-- Casting cover image: castings.cover_upload_id → uploads.id
-- The old cover_image_url string column is kept for backward compat during transition,
-- then removed by migration 000072.
ALTER TABLE castings
    ADD COLUMN IF NOT EXISTS cover_upload_id UUID
        REFERENCES uploads(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_castings_cover_upload_id
    ON castings(cover_upload_id)
    WHERE cover_upload_id IS NOT NULL;

COMMENT ON COLUMN model_profiles.avatar_upload_id IS
    '1:1 FK to uploads. NULL means no avatar set. On upload delete, avatar is cleared (SET NULL).';
COMMENT ON COLUMN castings.cover_upload_id IS
    '1:1 FK to uploads. Replaces the legacy cover_image_url string column.';
