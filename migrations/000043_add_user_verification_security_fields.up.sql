-- Adds missing columns referenced by application code for employer/company verification
-- and auth/security features. Uses IF NOT EXISTS so it can be applied safely.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS verification_notes TEXT,
    ADD COLUMN IF NOT EXISTS verification_rejection_reason TEXT,
    ADD COLUMN IF NOT EXISTS verification_submitted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS verification_reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS verification_reviewed_by UUID,
    ADD COLUMN IF NOT EXISTS verification_token TEXT,
    ADD COLUMN IF NOT EXISTS reset_token TEXT,
    ADD COLUMN IF NOT EXISTS reset_token_exp TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_login_ip TEXT,
    ADD COLUMN IF NOT EXISTS two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS two_factor_secret TEXT;
