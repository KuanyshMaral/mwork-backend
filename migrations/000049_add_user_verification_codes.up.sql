-- Ensure verification flag exists on users
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_verified BOOLEAN;
UPDATE users SET is_verified = FALSE WHERE is_verified IS NULL;
ALTER TABLE users ALTER COLUMN is_verified SET DEFAULT FALSE;
ALTER TABLE users ALTER COLUMN is_verified SET NOT NULL;

-- One active verification code per user (latest record per user)
CREATE TABLE IF NOT EXISTS user_verification_codes (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    code_hash VARCHAR(255) NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_verification_codes_user_id ON user_verification_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_user_verification_codes_expires_at ON user_verification_codes(expires_at);
