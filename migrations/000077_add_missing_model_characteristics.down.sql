-- Revert Migration 000077: Add missing model characteristics

ALTER TABLE model_profiles
    DROP COLUMN IF EXISTS bust_cm,
    DROP COLUMN IF EXISTS waist_cm,
    DROP COLUMN IF EXISTS hips_cm,
    DROP COLUMN IF EXISTS skin_tone;

-- Create a temporary function for safe jsonb to text[] casting
CREATE OR REPLACE FUNCTION pg_temp.jsonb_to_text_array(j jsonb) RETURNS text[] AS $$
BEGIN
    RETURN (SELECT array_agg(x) FROM jsonb_array_elements_text(j) x);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Revert specializations back to TEXT[]
ALTER TABLE model_profiles
    ALTER COLUMN specializations DROP DEFAULT,
    ALTER COLUMN specializations TYPE TEXT[] USING pg_temp.jsonb_to_text_array(specializations),
    ALTER COLUMN specializations DROP NOT NULL,
    ALTER COLUMN specializations SET DEFAULT '{}'::TEXT[];
