-- 000034_add_uploads_table.down.sql
-- Rollback uploads table

DROP INDEX IF EXISTS idx_uploads_status;
DROP INDEX IF EXISTS idx_uploads_user_id;

DROP TABLE IF EXISTS uploads;
