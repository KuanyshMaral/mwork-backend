-- Remove kaspi_order_id column from payments table
ALTER TABLE payments
    DROP COLUMN IF EXISTS kaspi_order_id;

-- Remove the unique index on kaspi_order_id
DROP INDEX IF EXISTS idx_payments_kaspi_order_id;
