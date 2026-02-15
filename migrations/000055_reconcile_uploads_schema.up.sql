-- Ensure uploads table matches the current upload domain schema.
-- This migration is backward-compatible with legacy installations where
-- uploads table was created with temp_path/final_path and pending/confirmed statuses.

ALTER TABLE uploads
    ADD COLUMN IF NOT EXISTS category VARCHAR(20),
    ADD COLUMN IF NOT EXISTS original_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS mime_type VARCHAR(100),
    ADD COLUMN IF NOT EXISTS size BIGINT,
    ADD COLUMN IF NOT EXISTS staging_key VARCHAR(500),
    ADD COLUMN IF NOT EXISTS permanent_key VARCHAR(500),
    ADD COLUMN IF NOT EXISTS permanent_url VARCHAR(1000),
    ADD COLUMN IF NOT EXISTS width INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS height INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS error_message TEXT,
    ADD COLUMN IF NOT EXISTS committed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;

-- Backfill values for legacy rows.
UPDATE uploads
SET category = COALESCE(category, 'photo'),
    original_name = COALESCE(original_name, 'legacy-upload'),
    mime_type = COALESCE(mime_type, 'application/octet-stream'),
    size = COALESCE(size, 1),
    staging_key = COALESCE(staging_key, temp_path),
    permanent_key = COALESCE(permanent_key, final_path),
    width = COALESCE(width, 0),
    height = COALESCE(height, 0),
    committed_at = COALESCE(committed_at, confirmed_at),
    expires_at = COALESCE(expires_at, created_at + INTERVAL '1 hour')
WHERE category IS NULL
   OR original_name IS NULL
   OR mime_type IS NULL
   OR size IS NULL
   OR staging_key IS NULL
   OR permanent_key IS NULL
   OR width IS NULL
   OR height IS NULL
   OR committed_at IS NULL
   OR expires_at IS NULL;

-- Normalize legacy statuses.
UPDATE uploads SET status = 'staged' WHERE status = 'pending';
UPDATE uploads SET status = 'committed' WHERE status = 'confirmed';

-- Ensure required defaults / not-nulls for active schema.
ALTER TABLE uploads
    ALTER COLUMN category SET DEFAULT 'photo',
    ALTER COLUMN category SET NOT NULL,
    ALTER COLUMN original_name SET NOT NULL,
    ALTER COLUMN mime_type SET NOT NULL,
    ALTER COLUMN size SET NOT NULL,
    ALTER COLUMN width SET DEFAULT 0,
    ALTER COLUMN height SET DEFAULT 0,
    ALTER COLUMN expires_at SET NOT NULL;

-- Replace incompatible legacy check constraints.
ALTER TABLE uploads DROP CONSTRAINT IF EXISTS uploads_status_check;
ALTER TABLE uploads DROP CONSTRAINT IF EXISTS uploads_category_check;
ALTER TABLE uploads DROP CONSTRAINT IF EXISTS staging_or_permanent;

ALTER TABLE uploads
    ADD CONSTRAINT uploads_status_check
        CHECK (status IN ('staged', 'committed', 'failed', 'deleted')),
    ADD CONSTRAINT uploads_category_check
        CHECK (category IN ('avatar', 'photo', 'document')),
    ADD CONSTRAINT staging_or_permanent
        CHECK (
            (status = 'staged' AND staging_key IS NOT NULL) OR
            (status = 'committed' AND permanent_key IS NOT NULL) OR
            (status IN ('failed', 'deleted'))
        );

-- Add missing indexes used by repository queries.
CREATE INDEX IF NOT EXISTS idx_uploads_user_category
    ON uploads(user_id, category) WHERE status = 'committed';
CREATE INDEX IF NOT EXISTS idx_uploads_expires
    ON uploads(expires_at) WHERE status = 'staged';
