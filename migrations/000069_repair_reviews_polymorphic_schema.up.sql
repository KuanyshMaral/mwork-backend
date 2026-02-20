-- Repair migration for environments where 000067 was skipped or partially applied.
-- Makes the reviews schema compatible with polymorphic review repository queries.

-- Remove legacy trigger/function if they still exist.
DROP TRIGGER IF EXISTS trigger_update_profile_rating ON reviews;
DROP FUNCTION IF EXISTS update_profile_rating();

DO $$
BEGIN
    -- profile_id -> target_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'reviews' AND column_name = 'profile_id'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'reviews' AND column_name = 'target_id'
    ) THEN
        ALTER TABLE reviews RENAME COLUMN profile_id TO target_id;
    END IF;

    -- reviewer_id -> author_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'reviews' AND column_name = 'reviewer_id'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'reviews' AND column_name = 'author_id'
    ) THEN
        ALTER TABLE reviews RENAME COLUMN reviewer_id TO author_id;
    END IF;

    -- casting_id -> context_id
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'reviews' AND column_name = 'casting_id'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'reviews' AND column_name = 'context_id'
    ) THEN
        ALTER TABLE reviews RENAME COLUMN casting_id TO context_id;
    END IF;
END $$;

ALTER TABLE reviews
    ADD COLUMN IF NOT EXISTS target_type VARCHAR(50) NOT NULL DEFAULT 'model_profile',
    ADD COLUMN IF NOT EXISTS context_type VARCHAR(50),
    ADD COLUMN IF NOT EXISTS criteria JSONB NOT NULL DEFAULT '{}';

-- Cleanup stale constraints/fks from legacy schema.
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_profile_id_fkey;
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_reviewer_id_fkey;
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_profile_id_reviewer_id_casting_id_key;

-- Recreate modern constraints.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'reviews_author_id_fkey'
    ) THEN
        ALTER TABLE reviews
            ADD CONSTRAINT reviews_author_id_fkey
            FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'reviews_unique_per_target'
    ) THEN
        ALTER TABLE reviews
            ADD CONSTRAINT reviews_unique_per_target
            UNIQUE (author_id, target_type, target_id, context_id);
    END IF;
END $$;

-- Ensure supporting indexes exist.
DROP INDEX IF EXISTS idx_reviews_profile;
CREATE INDEX IF NOT EXISTS idx_reviews_target ON reviews(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_reviews_author ON reviews(author_id);

-- Ensure cached stats columns exist on target tables.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'model_profiles' AND column_name = 'total_reviews'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'model_profiles' AND column_name = 'reviews_count'
    ) THEN
        ALTER TABLE model_profiles RENAME COLUMN total_reviews TO reviews_count;
    END IF;
END $$;

ALTER TABLE model_profiles
    ADD COLUMN IF NOT EXISTS rating_score FLOAT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reviews_count INT NOT NULL DEFAULT 0;

ALTER TABLE employer_profiles
    ADD COLUMN IF NOT EXISTS rating_score FLOAT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reviews_count INT NOT NULL DEFAULT 0;

ALTER TABLE castings
    ADD COLUMN IF NOT EXISTS rating_score FLOAT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reviews_count INT NOT NULL DEFAULT 0;