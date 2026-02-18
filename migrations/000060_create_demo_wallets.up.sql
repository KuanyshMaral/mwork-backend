CREATE TABLE user_wallets (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('topup', 'payment', 'refund')),
    reference_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_wallet_transactions_user ON wallet_transactions(user_id);
CREATE UNIQUE INDEX idx_wallet_transactions_ref_unique
    ON wallet_transactions(user_id, type, reference_id)
    WHERE reference_id IS NOT NULL;
