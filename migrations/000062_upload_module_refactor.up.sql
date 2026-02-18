-- Add new columns for extended functionality
ALTER TABLE uploads ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;
ALTER TABLE uploads ADD COLUMN IF NOT EXISTS batch_id UUID;
ALTER TABLE uploads ADD COLUMN IF NOT EXISTS purpose VARCHAR(50);

-- Add index for batch operations
CREATE INDEX IF NOT EXISTS idx_uploads_batch_id ON uploads(batch_id) WHERE batch_id IS NOT NULL;

-- Add index for metadata queries (GIN for JSONB)
CREATE INDEX IF NOT EXISTS idx_uploads_metadata ON uploads USING GIN (metadata);

-- Add check constraint for purpose validation (nullable for backward compat)
-- Drop first to allow re-run
ALTER TABLE uploads DROP CONSTRAINT IF EXISTS uploads_purpose_check;
ALTER TABLE uploads ADD CONSTRAINT uploads_purpose_check 
  CHECK (purpose IS NULL OR purpose IN (
    'avatar', 'photo', 'document', 
    'casting_cover', 'portfolio', 'chat_file', 
    'gallery', 'video', 'audio'
  ));

-- Update existing records: map category -> purpose
-- Idempotent because of WHERE clause
UPDATE uploads SET purpose = category WHERE purpose IS NULL;

-- Add comment for documentation
COMMENT ON COLUMN uploads.purpose IS 'File purpose/category - replaces legacy category field';
COMMENT ON COLUMN uploads.batch_id IS 'Groups multiple uploads together for batch operations';
COMMENT ON COLUMN uploads.metadata IS 'Custom metadata as JSON, e.g. {"album_id": "123", "order": 1}';
