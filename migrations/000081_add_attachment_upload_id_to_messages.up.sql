-- Ensure backward-compatible singular attachment reference exists for legacy clients/queries
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS attachment_upload_id UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'messages_attachment_upload_id_fkey'
    ) THEN
        ALTER TABLE messages
            ADD CONSTRAINT messages_attachment_upload_id_fkey
                FOREIGN KEY (attachment_upload_id)
                REFERENCES uploads(id)
                ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_messages_attachments
    ON messages(attachment_upload_id)
    WHERE attachment_upload_id IS NOT NULL;
