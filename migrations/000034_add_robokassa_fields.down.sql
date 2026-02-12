DROP INDEX IF EXISTS idx_payment_events_payment_id;
DROP TABLE IF EXISTS payment_events;
DROP INDEX IF EXISTS uq_payments_provider_robokassa_inv;
ALTER TABLE payments
    DROP COLUMN IF EXISTS raw_callback_payload,
    DROP COLUMN IF EXISTS raw_init_payload,
    DROP COLUMN IF EXISTS robokassa_inv_id;
