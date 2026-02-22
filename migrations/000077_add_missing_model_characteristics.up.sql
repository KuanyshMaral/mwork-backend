-- Migration 000077: Add missing model characteristics
-- Adds physical measurements and specializations that were mapped in code but missing in the DB schema.

ALTER TABLE model_profiles
    ADD COLUMN IF NOT EXISTS bust_cm INTEGER,
    ADD COLUMN IF NOT EXISTS waist_cm INTEGER,
    ADD COLUMN IF NOT EXISTS hips_cm INTEGER,
    ADD COLUMN IF NOT EXISTS skin_tone VARCHAR(50),
    ADD COLUMN IF NOT EXISTS specializations JSONB NOT NULL DEFAULT '[]'::jsonb;
