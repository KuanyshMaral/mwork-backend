-- Restore legacy Kaspi artifacts (rollback)
ALTER TABLE payments ADD COLUMN IF NOT EXISTS kaspi_order_id VARCHAR(100);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_kaspi_order_id ON payments(kaspi_order_id);
COMMENT ON COLUMN payments.kaspi_order_id IS 'Legacy Kaspi order ID';

CREATE TABLE IF NOT EXISTS kaspi_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID REFERENCES payments(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    raw_payload JSONB NOT NULL,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_kaspi_events_payment_id ON kaspi_events(payment_id);
CREATE INDEX IF NOT EXISTS idx_kaspi_events_event_type ON kaspi_events(event_type);
CREATE UNIQUE INDEX IF NOT EXISTS uq_kaspi_events_payment_type_created_at ON kaspi_events(payment_id, event_type, created_at);
