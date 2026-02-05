DROP FUNCTION IF EXISTS reconcile_user_credits(UUID);

DROP INDEX IF EXISTS idx_credit_transactions_user_created_at;

DROP TABLE IF EXISTS credit_transactions;
