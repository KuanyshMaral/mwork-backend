DROP INDEX IF EXISTS idx_messages_attachments;

ALTER TABLE messages
    DROP CONSTRAINT IF EXISTS messages_attachment_upload_id_fkey;

ALTER TABLE messages
    DROP COLUMN IF EXISTS attachment_upload_id;
