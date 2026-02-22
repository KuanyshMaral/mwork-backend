-- Migration 000077: Add missing model characteristics
-- Adds physical measurements and specializations that were mapped in code but missing in the DB schema.

ALTER TABLE model_profiles
    ADD COLUMN IF NOT EXISTS bust_cm INTEGER,
    ADD COLUMN IF NOT EXISTS waist_cm INTEGER,
    ADD COLUMN IF NOT EXISTS hips_cm INTEGER,
    ADD COLUMN IF NOT EXISTS skin_tone VARCHAR(50);

-- Use a DO block with dynamic SQL to handle environmental differences safely.
-- In some environments (like dev), the column exists as TEXT[] and needs converting.
-- In others (like staging/prod), it might not exist at all and needs adding.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name='model_profiles' AND column_name='specializations'
    ) THEN
        EXECUTE 'ALTER TABLE model_profiles ADD COLUMN specializations JSONB NOT NULL DEFAULT ''[]''::jsonb';
    ELSE
        -- The column exists, check if it needs type casting from text[] to jsonb
        IF EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name='model_profiles' AND column_name='specializations' AND data_type != 'jsonb'
        ) THEN
            -- Drop default first, otherwise PG fails to cast the default value '{text, array}'
            EXECUTE 'ALTER TABLE model_profiles ALTER COLUMN specializations DROP DEFAULT';
            EXECUTE 'ALTER TABLE model_profiles ALTER COLUMN specializations TYPE JSONB USING to_jsonb(specializations)';
            EXECUTE 'UPDATE model_profiles SET specializations = ''[]''::jsonb WHERE specializations IS NULL';
            EXECUTE 'ALTER TABLE model_profiles ALTER COLUMN specializations SET DEFAULT ''[]''::jsonb';
            EXECUTE 'ALTER TABLE model_profiles ALTER COLUMN specializations SET NOT NULL';
        END IF;
    END IF;
END $$;
