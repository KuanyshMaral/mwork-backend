DROP INDEX IF EXISTS uq_payments_inv_id;

ALTER TABLE payments
    DROP COLUMN IF EXISTS response_package,
    DROP COLUMN IF EXISTS inv_id,
    DROP COLUMN IF EXISTS plan,
    DROP COLUMN IF EXISTS type;
