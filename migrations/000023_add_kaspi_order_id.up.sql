-- Add kaspi_order_id column to payments table
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS kaspi_order_id VARCHAR(100);

-- Add unique index for kaspi_order_id
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_kaspi_order_id
    ON payments(kaspi_order_id);

-- Add comment to kaspi_order_id column
COMMENT ON COLUMN payments.kaspi_order_id
    IS 'Unique order ID from Kaspi payment gateway';
