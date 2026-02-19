ALTER TABLE plans
    ADD COLUMN IF NOT EXISTS audience VARCHAR(20);

UPDATE plans
SET audience = CASE
    WHEN id = 'agency' THEN 'employer'
    ELSE 'model'
END
WHERE audience IS NULL;

ALTER TABLE plans
    ALTER COLUMN audience SET NOT NULL;

ALTER TABLE plans
    ADD CONSTRAINT plans_audience_check CHECK (audience IN ('model', 'employer'));

CREATE INDEX IF NOT EXISTS idx_plans_audience_active ON plans(audience, is_active);

INSERT INTO plans (
    id, name, description, price_monthly, price_yearly,
    max_photos, max_responses_month, can_chat, can_see_viewers, priority_search, max_team_members,
    is_active, created_at, audience
)
SELECT
    'free_model',
    'Free Model',
    COALESCE(description, 'Базовый бесплатный план для моделей'),
    0,
    NULL,
    max_photos,
    max_responses_month,
    can_chat,
    can_see_viewers,
    priority_search,
    max_team_members,
    true,
    NOW(),
    'model'
FROM plans
WHERE id = 'free'
  AND NOT EXISTS (SELECT 1 FROM plans WHERE id = 'free_model');

INSERT INTO plans (
    id, name, description, price_monthly, price_yearly,
    max_photos, max_responses_month, can_chat, can_see_viewers, priority_search, max_team_members,
    is_active, created_at, audience
)
VALUES (
    'free_employer',
    'Free Employer',
    'Базовый бесплатный план для работодателей',
    0,
    NULL,
    10,
    0,
    true,
    false,
    false,
    1,
    true,
    NOW(),
    'employer'
)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS limit_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    limit_key TEXT NOT NULL,
    delta INTEGER NOT NULL,
    reason TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_limit_overrides_user_key ON limit_overrides(user_id, limit_key);
