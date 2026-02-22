-- Revert Migration 000077: Add missing model characteristics

ALTER TABLE model_profiles
    DROP COLUMN IF EXISTS bust_cm,
    DROP COLUMN IF EXISTS waist_cm,
    DROP COLUMN IF EXISTS hips_cm,
    DROP COLUMN IF EXISTS skin_tone,
    DROP COLUMN IF EXISTS specializations;
