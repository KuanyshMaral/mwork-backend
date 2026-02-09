-- Replace kaspi_order_id with robokassa_inv_id in payments table

-- First, drop the old kaspi_order_id column
ALTER TABLE payments DROP COLUMN IF EXISTS kaspi_order_id;

-- Add the new robokassa_inv_id column
ALTER TABLE payments ADD COLUMN robokassa_inv_id BIGINT;

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_payments_robokassa_inv_id ON payments(robokassa_inv_id);

-- Update provider column from 'kaspi' to 'robokassa' where applicable
UPDATE payments SET provider = 'robokassa' WHERE provider = 'kaspi';
