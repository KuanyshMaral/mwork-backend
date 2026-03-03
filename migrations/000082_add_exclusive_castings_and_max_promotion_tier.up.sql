-- Add exclusive castings visibility flag
ALTER TABLE castings
    ADD COLUMN IF NOT EXISTS is_exclusive BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_castings_is_exclusive_active
    ON castings(is_exclusive, status)
    WHERE status = 'active';

-- Add max-tier marker for profile promotion slot prioritization
ALTER TABLE profile_promotions
    ADD COLUMN IF NOT EXISTS is_max_tier BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_profile_promotions_active_tier_budget
    ON profile_promotions(status, is_max_tier, budget_amount DESC)
    WHERE status = 'active';

-- Plan lineup update: free/start/pro (+ keep legacy free_model as inactive)
INSERT INTO plans (
    id, name, description, price_monthly, price_yearly,
    audience, is_active, created_at,
    monthly_consumables, features_and_quotas
)
SELECT
    'free',
    'Free',
    COALESCE(description, 'Базовый бесплатный план для моделей'),
    0,
    NULL,
    'model',
    true,
    NOW(),
    COALESCE(monthly_consumables, '{"response_connects":20}'::jsonb),
    COALESCE(features_and_quotas, '{"max_photos":3,"can_chat":true,"can_see_viewers":false,"priority_search":false,"max_team_members":0,"max_active_castings":0}'::jsonb)
FROM plans
WHERE id = 'free_model'
  AND NOT EXISTS (SELECT 1 FROM plans WHERE id = 'free');

UPDATE subscriptions
SET plan_id = 'free'
WHERE plan_id = 'free_model'
  AND EXISTS (SELECT 1 FROM plans WHERE id = 'free');

INSERT INTO plans (
    id, name, description, price_monthly, price_yearly,
    audience, is_active, created_at,
    monthly_consumables, features_and_quotas
)
VALUES (
    'start',
    'MWork Start',
    'Стартовый платный план для моделей',
    4990,
    NULL,
    'model',
    true,
    NOW(),
    '{"response_connects":100}'::jsonb,
    '{"max_photos":100,"can_chat":true,"can_see_viewers":true,"can_view_exclusive_castings":true,"priority_search":true,"promotion_tier":"max","max_team_members":0,"max_active_castings":0}'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    price_monthly = EXCLUDED.price_monthly,
    audience = EXCLUDED.audience,
    is_active = true,
    monthly_consumables = EXCLUDED.monthly_consumables,
    features_and_quotas = EXCLUDED.features_and_quotas;

UPDATE plans
SET
    name = 'MWork Pro',
    description = 'Профессиональный план для моделей',
    price_monthly = 9990,
    audience = 'model',
    is_active = true,
    monthly_consumables = '{"response_connects":-1}'::jsonb,
    features_and_quotas = '{"max_photos":100,"can_chat":true,"can_see_viewers":true,"can_view_exclusive_castings":true,"priority_search":true,"promotion_tier":"max","max_team_members":0,"max_active_castings":0}'::jsonb
WHERE id = 'pro';

UPDATE plans
SET
    name = 'Free',
    is_active = true,
    monthly_consumables = jsonb_set(COALESCE(monthly_consumables, '{}'::jsonb), '{response_connects}', '20'::jsonb, true),
    features_and_quotas = jsonb_set(
        jsonb_set(COALESCE(features_and_quotas, '{}'::jsonb), '{can_view_exclusive_castings}', 'false'::jsonb, true),
        '{promotion_tier}',
        '"none"'::jsonb,
        true
    )
WHERE id = 'free';

UPDATE plans
SET
    is_active = false,
    features_and_quotas = jsonb_set(
        jsonb_set(COALESCE(features_and_quotas, '{}'::jsonb), '{can_view_exclusive_castings}', 'false'::jsonb, true),
        '{promotion_tier}',
        '"none"'::jsonb,
        true
    )
WHERE id = 'free_model';

UPDATE plans
SET features_and_quotas = jsonb_set(
        jsonb_set(COALESCE(features_and_quotas, '{}'::jsonb), '{can_view_exclusive_castings}', 'false'::jsonb, true),
        '{promotion_tier}',
        '"none"'::jsonb,
        true
    )
WHERE id IN ('free_employer', 'agency')
  AND NOT (features_and_quotas ? 'promotion_tier');
