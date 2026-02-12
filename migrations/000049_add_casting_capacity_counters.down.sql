ALTER TABLE castings
    DROP CONSTRAINT IF EXISTS castings_accepted_models_not_exceed_required,
    DROP CONSTRAINT IF EXISTS castings_accepted_models_count_non_negative,
    DROP CONSTRAINT IF EXISTS castings_required_models_count_non_negative;

ALTER TABLE castings
    DROP COLUMN IF EXISTS accepted_models_count,
    DROP COLUMN IF EXISTS required_models_count;
