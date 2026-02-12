ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS robokassa_inv_id BIGINT,
    ADD COLUMN IF NOT EXISTS raw_init_payload JSONB,
    ADD COLUMN IF NOT EXISTS raw_callback_payload JSONB;

CREATE UNIQUE INDEX IF NOT EXISTS uq_payments_provider_robokassa_inv
    ON payments(provider, robokassa_inv_id)
    WHERE robokassa_inv_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS payment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_events_payment_id ON payment_events(payment_id, created_at DESC);
