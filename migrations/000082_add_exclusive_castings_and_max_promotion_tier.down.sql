-- Revert new flags
ALTER TABLE castings DROP COLUMN IF EXISTS is_exclusive;
ALTER TABLE profile_promotions DROP COLUMN IF EXISTS is_max_tier;

-- Revert model plan lineup (best-effort)
UPDATE subscriptions
SET plan_id = 'free_model'
WHERE plan_id = 'free'
  AND EXISTS (SELECT 1 FROM plans WHERE id = 'free_model');

DELETE FROM plans WHERE id = 'start';

UPDATE plans
SET
    name = 'Pro',
    price_monthly = 3990
WHERE id = 'pro';

UPDATE plans
SET is_active = true
WHERE id = 'free_model';
