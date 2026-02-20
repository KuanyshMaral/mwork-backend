-- Rollback 000066: Restore original reviews table structure
ALTER TABLE castings DROP COLUMN IF EXISTS rating_score, DROP COLUMN IF EXISTS reviews_count;
ALTER TABLE employer_profiles DROP COLUMN IF EXISTS rating_score, DROP COLUMN IF EXISTS reviews_count;
ALTER TABLE model_profiles DROP COLUMN IF EXISTS rating_score;
ALTER TABLE model_profiles RENAME COLUMN IF EXISTS reviews_count TO total_reviews;

DROP INDEX IF EXISTS idx_reviews_target;
DROP INDEX IF EXISTS idx_reviews_author;

ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_unique_per_target;
ALTER TABLE reviews DROP CONSTRAINT IF EXISTS reviews_author_id_fkey;
ALTER TABLE reviews RENAME COLUMN context_id TO casting_id;
ALTER TABLE reviews RENAME COLUMN author_id TO reviewer_id;
ALTER TABLE reviews DROP COLUMN IF EXISTS criteria;
ALTER TABLE reviews DROP COLUMN IF EXISTS context_type;
ALTER TABLE reviews DROP COLUMN IF EXISTS target_type;
ALTER TABLE reviews RENAME COLUMN target_id TO profile_id;
