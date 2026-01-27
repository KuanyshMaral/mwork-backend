-- Migration: Add yearly billing option to subscriptions
-- Purpose: Support annual subscriptions with discount

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS billing_period VARCHAR(20) DEFAULT 'monthly';
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS yearly_discount_percent INT DEFAULT 17;

-- Add yearly prices to plans (already has price_yearly column)
-- Just ensure it's populated
UPDATE plans
SET price_yearly = ROUND(price_monthly * 12 * 0.83)
WHERE price_yearly IS NULL;

-- Index for billing period queries
CREATE INDEX IF NOT EXISTS idx_subscriptions_billing ON subscriptions(billing_period);

-- Comment
COMMENT ON COLUMN subscriptions.billing_period IS 'monthly or yearly';
COMMENT ON COLUMN subscriptions.yearly_discount_percent IS 'Discount percentage for yearly billing';
