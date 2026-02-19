DROP INDEX IF EXISTS idx_limit_overrides_user_key;
DROP TABLE IF EXISTS limit_overrides;

DELETE FROM plans WHERE id IN ('free_model', 'free_employer');

DROP INDEX IF EXISTS idx_plans_audience_active;
ALTER TABLE plans DROP CONSTRAINT IF EXISTS plans_audience_check;
ALTER TABLE plans DROP COLUMN IF EXISTS audience;
