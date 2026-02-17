-- Drop indexes
DROP INDEX IF EXISTS idx_uploads_metadata;
DROP INDEX IF EXISTS idx_uploads_batch_id;

-- Drop constraint
ALTER TABLE uploads DROP CONSTRAINT IF EXISTS uploads_purpose_check;

-- Drop columns
ALTER TABLE uploads DROP COLUMN IF EXISTS purpose;
ALTER TABLE uploads DROP COLUMN IF EXISTS batch_id;
ALTER TABLE uploads DROP COLUMN IF EXISTS metadata;
