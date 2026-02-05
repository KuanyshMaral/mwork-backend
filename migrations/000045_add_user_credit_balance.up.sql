ALTER TABLE users
ADD COLUMN credit_balance INTEGER NOT NULL DEFAULT 0
    CHECK (credit_balance >= 0);

CREATE INDEX idx_users_credit_balance ON users (credit_balance);
