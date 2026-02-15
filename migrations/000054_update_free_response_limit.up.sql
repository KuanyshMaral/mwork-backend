-- 000054_update_free_response_limit.up.sql
-- Increase free plan monthly response limit for models

UPDATE plans
SET max_responses_month = 20
WHERE id = 'free';