-- Rollback: Remove organizations and verification fields

DROP INDEX IF EXISTS idx_users_verification;
DROP INDEX IF EXISTS idx_users_org;
DROP INDEX IF EXISTS idx_organizations_created;
DROP INDEX IF EXISTS idx_organizations_status;
DROP INDEX IF EXISTS idx_organizations_bin;

ALTER TABLE users DROP COLUMN IF EXISTS user_verification_status;
ALTER TABLE users DROP COLUMN IF EXISTS organization_id;

DROP TABLE IF EXISTS organizations;

DROP TYPE IF EXISTS verification_status;
DROP TYPE IF EXISTS org_type;
