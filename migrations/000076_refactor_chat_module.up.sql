-- 000076_refactor_chat_module.up.sql
-- Drop the singular attachment_upload_id column from messages
-- as we are migrating to the polymorphic attachments table 
-- with target_type = 'chat_attachment'.

ALTER TABLE messages DROP COLUMN IF EXISTS attachment_upload_id;
