-- Remove legacy Kaspi payment artifacts (project switched to Robokassa only)
DROP INDEX IF EXISTS idx_payments_kaspi_order_id;
ALTER TABLE payments DROP COLUMN IF EXISTS kaspi_order_id;

DROP INDEX IF EXISTS uq_kaspi_events_payment_type_created_at;
DROP INDEX IF EXISTS idx_kaspi_events_event_type;
DROP INDEX IF EXISTS idx_kaspi_events_payment_id;
DROP TABLE IF EXISTS kaspi_events;
