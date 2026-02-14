-- 000044_add_user_status.down.sql
-- Remove status columns from users table

-- Drop indexes
DROP INDEX IF EXISTS idx_users_role_status;
DROP INDEX IF EXISTS idx_users_status;

-- Drop columns
ALTER TABLE users DROP COLUMN IF EXISTS status_updated_at;
ALTER TABLE users DROP COLUMN IF EXISTS status_reason;
ALTER TABLE users DROP COLUMN IF EXISTS status;
