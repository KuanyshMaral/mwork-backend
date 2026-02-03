-- 000010_create_admin.up.sql
-- Admin users, audit logs, and feature flags

-- Separate table for admin users (not mixed with regular users)
CREATE TABLE admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    
    -- Role: super_admin, admin, moderator, support
    role VARCHAR(20) NOT NULL DEFAULT 'support',
    
    -- Metadata
    name VARCHAR(100) NOT NULL,
    avatar_url VARCHAR(500),
    
    -- Security
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    last_login_ip VARCHAR(45),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Audit logs for all admin actions
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Who performed the action
    admin_id UUID REFERENCES admin_users(id),
    admin_email VARCHAR(255), -- Denormalized for history
    
    -- What action was performed
    action VARCHAR(50) NOT NULL, -- user.ban, casting.delete, subscription.upgrade
    entity_type VARCHAR(50) NOT NULL, -- user, casting, profile, photo, etc.
    entity_id UUID,
    
    -- Details
    old_value JSONB,
    new_value JSONB,
    reason TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Feature flags for runtime configuration
CREATE TABLE feature_flags (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL DEFAULT 'true',
    description TEXT,
    updated_by UUID REFERENCES admin_users(id),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_admin_users_email ON admin_users(email);
CREATE INDEX idx_admin_users_role ON admin_users(role) WHERE is_active = true;

CREATE INDEX idx_audit_logs_admin ON audit_logs(admin_id, created_at DESC);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);

-- Default super admin (password: admin123 - CHANGE IN PRODUCTION!)
INSERT INTO admin_users (email, password_hash, role, name) VALUES
('admin@mwork.kz', '$2a$12$ob/WOA675I7wHxM/a9aQKexapuOA2ll28eCwHZhG8dxdEUEWL8kH.', 'super_admin', 'System Admin');

-- Default feature flags
INSERT INTO feature_flags (key, value, description) VALUES
('registration_enabled', 'true', 'Allow new user registrations'),
('photo_upload_enabled', 'true', 'Allow photo uploads'),
('chat_enabled', 'true', 'Enable chat functionality'),
('payments_enabled', 'false', 'Enable real payment processing'),
('max_photos_free', '3', 'Max photos for free tier users'),
('max_responses_free', '5', 'Max monthly responses for free tier');

-- Comments
COMMENT ON TABLE admin_users IS 'Admin panel users (separate from regular users)';
COMMENT ON TABLE audit_logs IS 'Audit trail of all admin actions';
COMMENT ON TABLE feature_flags IS 'Runtime feature configuration';
