ALTER TABLE users
ADD COLUMN IF NOT EXISTS credit_balance INTEGER NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_credit_balance_nonneg'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_credit_balance_nonneg CHECK (credit_balance >= 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_users_credit_balance ON users (credit_balance);
