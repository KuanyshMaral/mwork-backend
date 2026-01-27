-- migrations/000004_create_responses.down.sql

DROP TRIGGER IF EXISTS responses_count_trigger ON responses;
DROP FUNCTION IF EXISTS update_casting_response_count();
DROP TRIGGER IF EXISTS responses_updated_at ON responses;
DROP TABLE IF EXISTS responses;
