-- Reverts columns added in 000041_add_user_verification_security_fields.up.sql

ALTER TABLE users
DROP COLUMN IF EXISTS two_factor_secret,
  DROP COLUMN IF EXISTS two_factor_enabled,
  DROP COLUMN IF EXISTS last_login_ip,
  DROP COLUMN IF EXISTS last_login_at,
  DROP COLUMN IF EXISTS reset_token_exp,
  DROP COLUMN IF EXISTS reset_token,
  DROP COLUMN IF EXISTS verification_token,
  DROP COLUMN IF EXISTS verification_reviewed_by,
  DROP COLUMN IF EXISTS verification_reviewed_at,
  DROP COLUMN IF EXISTS verification_submitted_at,
  DROP COLUMN IF EXISTS verification_rejection_reason,
  DROP COLUMN IF EXISTS verification_notes;
