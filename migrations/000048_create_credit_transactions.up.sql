CREATE TABLE credit_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    amount_delta INTEGER NOT NULL,
    tx_type TEXT NOT NULL CHECK (tx_type IN ('deduction', 'refund', 'purchase', 'admin_grant')),
    related_entity_type TEXT NULL,
    related_entity_id UUID NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_credit_transactions_user_created_at
    ON credit_transactions (user_id, created_at DESC);

CREATE OR REPLACE FUNCTION reconcile_user_credits(p_user_id UUID)
RETURNS INTEGER
LANGUAGE SQL
STABLE
AS $$
    SELECT COALESCE(SUM(ct.amount_delta), 0)::INTEGER
    FROM credit_transactions ct
    WHERE ct.user_id = p_user_id;
$$;
