-- Migration 000075: Refactor `plans` table to use JSONB for extensibility
-- and add model profile fields + model connects to users.

-- ============================================================
-- Step 1: Add JSONB columns to `plans` (Camp 1 & Camp 2)
-- ============================================================
ALTER TABLE plans
    ADD COLUMN IF NOT EXISTS monthly_consumables  JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS features_and_quotas  JSONB NOT NULL DEFAULT '{}'::jsonb;

-- ============================================================
-- Step 2: Migrate existing sparse column data into JSONB
-- ============================================================

-- Model plans: monthly_consumables gets response_connects
UPDATE plans
SET monthly_consumables = jsonb_build_object('response_connects', max_responses_month)
WHERE audience = 'model' AND max_responses_month > 0;

-- All plans: features_and_quotas gets the Camp 2 data
UPDATE plans
SET features_and_quotas = jsonb_build_object(
    'max_photos',          COALESCE(max_photos, 0),
    'can_chat',            COALESCE(can_chat, false),
    'can_see_viewers',     COALESCE(can_see_viewers, false),
    'priority_search',     COALESCE(priority_search, false),
    'max_team_members',    COALESCE(max_team_members, 0),
    'max_active_castings', CASE WHEN audience = 'employer' THEN 3 ELSE 0 END
);

-- ============================================================
-- Step 3: Drop the old sparse columns we now no longer need
-- ============================================================
ALTER TABLE plans
    DROP COLUMN IF EXISTS max_photos,
    DROP COLUMN IF EXISTS max_responses_month,
    DROP COLUMN IF EXISTS can_chat,
    DROP COLUMN IF EXISTS can_see_viewers,
    DROP COLUMN IF EXISTS priority_search,
    DROP COLUMN IF EXISTS max_team_members;

-- ============================================================
-- Step 4: Add Model Connects (Two Buckets) to `users`
-- ============================================================
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS model_free_response_connects       INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS model_purchased_response_connects  INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS model_connects_reset_at            TIMESTAMPTZ;

-- Give free_model users their initial 20 free connects
UPDATE users u
SET 
    model_free_response_connects = 20,
    model_connects_reset_at = NOW()
WHERE u.role = 'model'
  AND u.model_free_response_connects = 0
  AND u.model_connects_reset_at IS NULL;

-- ============================================================
-- Step 5: Add missing model profile characteristics
-- ============================================================
ALTER TABLE model_profiles
    ADD COLUMN IF NOT EXISTS bust_cm          INTEGER,
    ADD COLUMN IF NOT EXISTS waist_cm         INTEGER,
    ADD COLUMN IF NOT EXISTS hips_cm          INTEGER,
    ADD COLUMN IF NOT EXISTS skin_tone        VARCHAR(50),
    ADD COLUMN IF NOT EXISTS specializations  JSONB NOT NULL DEFAULT '[]'::jsonb;
