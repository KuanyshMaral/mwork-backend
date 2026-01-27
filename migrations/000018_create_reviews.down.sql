-- Rollback: Drop reviews system

DROP TRIGGER IF EXISTS trigger_update_profile_rating ON reviews;
DROP FUNCTION IF EXISTS update_profile_rating();
DROP TABLE IF EXISTS reviews;
