-- 000009_create_payments.up.sql
-- Payment records for tracking transactions

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    
    -- Amount
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'KZT',
    
    -- Status
    status VARCHAR(20) DEFAULT 'pending', -- pending, completed, failed, refunded
    
    -- Payment provider
    provider VARCHAR(50), -- kaspi, card, manual
    external_id VARCHAR(100), -- Payment ID from provider
    
    -- Metadata
    description TEXT,
    metadata JSONB,
    
    -- Timestamps
    paid_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    refunded_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_payments_user ON payments(user_id, created_at DESC);
CREATE INDEX idx_payments_subscription ON payments(subscription_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_external ON payments(provider, external_id);

-- Comments
COMMENT ON TABLE payments IS 'Payment transaction records';
COMMENT ON COLUMN payments.external_id IS 'Transaction ID from payment provider (Kaspi, etc)';
