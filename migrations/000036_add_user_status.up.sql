-- 000036_add_user_status.up.sql
-- Add status column to users table for employer moderation

-- Add status column to users table
ALTER TABLE users ADD COLUMN status VARCHAR(20) DEFAULT 'active' 
    CHECK (status IN ('active', 'pending', 'rejected', 'suspended'));

-- Add status_reason column for rejection/suspension reasons
ALTER TABLE users ADD COLUMN status_reason TEXT;

-- Add status_updated_at column to track when status was last changed
ALTER TABLE users ADD COLUMN status_updated_at TIMESTAMPTZ DEFAULT NOW();

-- Add index for status queries
CREATE INDEX idx_users_status ON users(status);

-- Add index for status and role combination (useful for admin queries)
CREATE INDEX idx_users_role_status ON users(role, status);

-- Update existing employers to have 'active' status (for existing data)
UPDATE users SET status = 'active' WHERE role = 'employer' AND status IS NULL;

-- Comments
COMMENT ON COLUMN users.status IS 'User status: active, pending, rejected, suspended';
COMMENT ON COLUMN users.status_reason IS 'Reason for status change (rejection/suspension)';
COMMENT ON COLUMN users.status_updated_at IS 'When status was last updated';
