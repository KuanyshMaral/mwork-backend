-- 000033_create_kaspi_events.down.sql
-- Rollback kaspi_events table

DROP INDEX IF EXISTS uq_kaspi_events_payment_type_created_at;
DROP INDEX IF EXISTS idx_kaspi_events_event_type;
DROP INDEX IF EXISTS idx_kaspi_events_payment_id;

DROP TABLE IF EXISTS kaspi_events;
