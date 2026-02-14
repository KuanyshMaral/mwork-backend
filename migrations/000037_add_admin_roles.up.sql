-- 000035_add_admin_roles.up.sql
-- Create mapping table from regular users -> admin permissions

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS user_admin_permissions (
                                                      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                                      user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                                      can_moderate BOOLEAN NOT NULL DEFAULT FALSE,
                                                      can_view_reports BOOLEAN NOT NULL DEFAULT FALSE,
                                                      can_manage_users BOOLEAN NOT NULL DEFAULT FALSE,
                                                      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_admin_permissions_user_id
    ON user_admin_permissions(user_id);

COMMENT ON TABLE user_admin_permissions IS 'Admin permissions mapping for regular users';
