-- 000076_refactor_chat_module.down.sql
ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_upload_id UUID REFERENCES uploads(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_messages_attachments ON messages(attachment_upload_id) WHERE attachment_upload_id IS NOT NULL;
