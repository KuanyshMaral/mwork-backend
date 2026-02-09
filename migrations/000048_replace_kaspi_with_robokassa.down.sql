-- Rollback: Replace robokassa_inv_id with kaspi_order_id

-- Drop the robokassa_inv_id column and index
DROP INDEX IF EXISTS idx_payments_robokassa_inv_id;
ALTER TABLE payments DROP COLUMN IF EXISTS robokassa_inv_id;

-- Add back the kaspi_order_id column
ALTER TABLE payments ADD COLUMN kaspi_order_id VARCHAR(255);

-- Revert provider column from 'robokassa' back to 'kaspi' where applicable
UPDATE payments SET provider = 'kaspi' WHERE provider = 'robokassa';
