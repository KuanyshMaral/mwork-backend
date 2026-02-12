ALTER TABLE castings
    ADD COLUMN required_models_count INTEGER NULL,
    ADD COLUMN accepted_models_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE castings
    ADD CONSTRAINT castings_required_models_count_non_negative
        CHECK (required_models_count IS NULL OR required_models_count >= 0),
    ADD CONSTRAINT castings_accepted_models_count_non_negative
        CHECK (accepted_models_count >= 0),
    ADD CONSTRAINT castings_accepted_models_not_exceed_required
        CHECK (required_models_count IS NULL OR accepted_models_count <= required_models_count);
