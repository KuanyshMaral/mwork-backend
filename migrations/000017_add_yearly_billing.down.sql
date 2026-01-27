-- Rollback: Remove yearly billing

DROP INDEX IF EXISTS idx_subscriptions_billing;

ALTER TABLE subscription_plans DROP COLUMN IF EXISTS yearly_price;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS yearly_discount_percent;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS billing_period;
