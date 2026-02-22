-- Migration 000077: Add missing model characteristics
-- Adds physical measurements and specializations that were mapped in code but missing in the DB schema.

ALTER TABLE model_profiles
    ADD COLUMN IF NOT EXISTS bust_cm INTEGER,
    ADD COLUMN IF NOT EXISTS waist_cm INTEGER,
    ADD COLUMN IF NOT EXISTS hips_cm INTEGER,
    ADD COLUMN IF NOT EXISTS skin_tone VARCHAR(50);

-- The specializations column was originally created as TEXT[] in migration 000014.
-- Since the codebase now expects it to be JSONB, we must explicitly ALTER its type.
-- to_jsonb() safely converts a Postgres text array to a JSON array.
ALTER TABLE model_profiles
    ALTER COLUMN specializations TYPE JSONB USING to_jsonb(specializations),
    ALTER COLUMN specializations SET DEFAULT '[]'::jsonb;

-- Also ensure it is NOT NULL if needed (though it might already be)
UPDATE model_profiles SET specializations = '[]'::jsonb WHERE specializations IS NULL;
ALTER TABLE model_profiles ALTER COLUMN specializations SET NOT NULL;
