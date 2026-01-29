ALTER TABLE uploads
    ADD COLUMN IF NOT EXISTS processed_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS process_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS process_attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS process_error TEXT NULL;

DO $$
    BEGIN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'uploads_process_status_check'
        ) THEN
            ALTER TABLE uploads
                ADD CONSTRAINT uploads_process_status_check
                    CHECK (process_status IN ('pending', 'processing', 'done', 'failed'));
        END IF;
    END$$;

CREATE INDEX IF NOT EXISTS idx_uploads_process_status
    ON uploads(process_status);

CREATE INDEX IF NOT EXISTS idx_uploads_status_process
    ON uploads(status, process_status);
