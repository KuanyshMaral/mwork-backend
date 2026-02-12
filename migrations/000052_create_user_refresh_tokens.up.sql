CREATE TABLE IF NOT EXISTS user_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    jti VARCHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_agent TEXT NOT NULL DEFAULT '',
    ip VARCHAR(45) NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_user_id ON user_refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_user_refresh_tokens_expires_at ON user_refresh_tokens(expires_at);
