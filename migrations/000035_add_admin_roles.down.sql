-- 000035_add_admin_roles.down.sql
-- Rollback mapping table

DROP INDEX IF EXISTS idx_user_admin_permissions_user_id;
DROP TABLE IF EXISTS user_admin_permissions;
