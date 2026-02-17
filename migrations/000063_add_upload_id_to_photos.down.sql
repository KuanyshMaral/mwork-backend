-- Remove upload_id from photos table
DROP INDEX IF EXISTS idx_photos_upload_id;
ALTER TABLE photos DROP COLUMN IF EXISTS upload_id;
