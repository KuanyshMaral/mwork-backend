-- 000033_create_kaspi_events.up.sql
-- Create kaspi_events table for webhook event logging
-- Used for idempotency and debugging of Kaspi webhooks

CREATE TABLE IF NOT EXISTS kaspi_events (
                                            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                            payment_id UUID REFERENCES payments(id) ON DELETE CASCADE,
                                            event_type VARCHAR(50) NOT NULL,
                                            raw_payload JSONB NOT NULL,
                                            processed BOOLEAN NOT NULL DEFAULT FALSE,
                                            created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_kaspi_events_payment_id
    ON kaspi_events(payment_id);

CREATE INDEX IF NOT EXISTS idx_kaspi_events_event_type
    ON kaspi_events(event_type);

-- Composite unique index for duplicate detection (best-effort)
CREATE UNIQUE INDEX IF NOT EXISTS uq_kaspi_events_payment_type_created_at
    ON kaspi_events(payment_id, event_type, created_at);

COMMENT ON TABLE kaspi_events
    IS 'Kaspi webhook event log for idempotency and debugging';
