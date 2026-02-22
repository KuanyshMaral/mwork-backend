-- Rollback for migration 000078

-- Restore model_profiles columns
ALTER TABLE model_profiles
    DROP COLUMN IF EXISTS specializations,
    DROP COLUMN IF EXISTS skin_tone,
    DROP COLUMN IF EXISTS hips_cm,
    DROP COLUMN IF EXISTS waist_cm,
    DROP COLUMN IF EXISTS bust_cm;

-- Restore users connects columns
ALTER TABLE users
    DROP COLUMN IF EXISTS model_connects_reset_at,
    DROP COLUMN IF EXISTS model_purchased_response_connects,
    DROP COLUMN IF EXISTS model_free_response_connects;

-- Restore plans sparse columns
ALTER TABLE plans
    ADD COLUMN IF NOT EXISTS max_photos          INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS max_responses_month INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS can_chat            BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS can_see_viewers     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS priority_search     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS max_team_members    INTEGER NOT NULL DEFAULT 0;

-- Attempt to restore data from JSONB columns back into sparse columns
UPDATE plans
SET
    max_photos          = COALESCE((features_and_quotas->>'max_photos')::int, 0),
    max_responses_month = COALESCE((monthly_consumables->>'response_connects')::int, 0),
    can_chat            = COALESCE((features_and_quotas->>'can_chat')::boolean, false),
    can_see_viewers     = COALESCE((features_and_quotas->>'can_see_viewers')::boolean, false),
    priority_search     = COALESCE((features_and_quotas->>'priority_search')::boolean, false),
    max_team_members    = COALESCE((features_and_quotas->>'max_team_members')::int, 0);

-- Drop JSONB columns
ALTER TABLE plans
    DROP COLUMN IF EXISTS monthly_consumables,
    DROP COLUMN IF EXISTS features_and_quotas;
