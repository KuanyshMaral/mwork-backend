-- ============================================================
-- Migration 000066: Refactor reviews to polymorphic table
-- ============================================================

-- Step 1: Drop legacy trigger and function
DROP TRIGGER IF EXISTS trigger_update_profile_rating ON reviews;
DROP FUNCTION IF EXISTS update_profile_rating();

-- Step 2: Refactor reviews table
ALTER TABLE reviews
    RENAME COLUMN profile_id TO target_id;

ALTER TABLE reviews
    ADD COLUMN IF NOT EXISTS target_type VARCHAR(50) NOT NULL DEFAULT 'model_profile';

ALTER TABLE reviews
    ADD COLUMN IF NOT EXISTS context_type VARCHAR(50);

ALTER TABLE reviews
    ADD COLUMN IF NOT EXISTS criteria JSONB NOT NULL DEFAULT '{}';

-- Rename reviewer_id -> author_id
ALTER TABLE reviews
    RENAME COLUMN reviewer_id TO author_id;

-- Drop the old FK on profile_id (now target_id — no longer FK since polymorphic)
ALTER TABLE reviews
    DROP CONSTRAINT IF EXISTS reviews_profile_id_fkey;

-- Drop the old FK on reviewer_id
ALTER TABLE reviews
    DROP CONSTRAINT IF EXISTS reviews_reviewer_id_fkey;

-- Re-add author FK on users (it had this FK as reviewer_id)
ALTER TABLE reviews
    ADD CONSTRAINT reviews_author_id_fkey
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;

-- Rename existing casting_id column to context_id (it was already the context)
ALTER TABLE reviews
    RENAME COLUMN casting_id TO context_id;

-- Drop old unique constraint
ALTER TABLE reviews
    DROP CONSTRAINT IF EXISTS reviews_profile_id_reviewer_id_casting_id_key;

-- New unique: one review per author per target + context
ALTER TABLE reviews
    ADD CONSTRAINT reviews_unique_per_target
    UNIQUE (author_id, target_type, target_id, context_id);

-- Step 3: Add indexes for polymorphic queries
DROP INDEX IF EXISTS idx_reviews_profile;
CREATE INDEX IF NOT EXISTS idx_reviews_target ON reviews(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_reviews_author ON reviews(author_id);

-- Step 4: Add cached stats counters to target tables

-- model_profiles: rename total_reviews -> reviews_count, add rating_score
ALTER TABLE model_profiles
    RENAME COLUMN total_reviews TO reviews_count;
ALTER TABLE model_profiles
    ADD COLUMN IF NOT EXISTS rating_score FLOAT NOT NULL DEFAULT 0;

-- employer_profiles: add counters
ALTER TABLE employer_profiles
    ADD COLUMN IF NOT EXISTS rating_score FLOAT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reviews_count INT NOT NULL DEFAULT 0;

-- castings: add counters
ALTER TABLE castings
    ADD COLUMN IF NOT EXISTS rating_score FLOAT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reviews_count INT NOT NULL DEFAULT 0;

-- Comments
COMMENT ON TABLE reviews IS 'Polymorphic review system: models, employers, castings';
COMMENT ON COLUMN reviews.target_type IS 'Reviewable entity type: model_profile, employer_profile, casting';
COMMENT ON COLUMN reviews.target_id IS 'UUID of the reviewed entity';
COMMENT ON COLUMN reviews.context_type IS 'Optional context type, e.g. casting';
COMMENT ON COLUMN reviews.context_id IS 'Optional context ID, e.g. casting_id';
COMMENT ON COLUMN reviews.criteria IS 'Specialized criteria scores as JSONB, e.g. {"punctuality": 5}';
