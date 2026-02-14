ALTER TABLE uploads
    DROP CONSTRAINT IF EXISTS uploads_process_status_check;

DROP INDEX IF EXISTS idx_uploads_process_status;
DROP INDEX IF EXISTS idx_uploads_status_process;

ALTER TABLE uploads
    DROP COLUMN IF EXISTS processed_at,
    DROP COLUMN IF EXISTS process_status,
    DROP COLUMN IF EXISTS process_attempts,
    DROP COLUMN IF EXISTS process_error;
