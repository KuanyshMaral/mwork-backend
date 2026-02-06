DROP INDEX IF EXISTS idx_users_credit_balance;

ALTER TABLE users
DROP COLUMN IF EXISTS credit_balance;
