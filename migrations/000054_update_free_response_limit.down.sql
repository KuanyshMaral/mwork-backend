-- 000054_update_free_response_limit.down.sql
-- Roll back free plan monthly response limit for models

UPDATE plans
SET max_responses_month = 5
WHERE id = 'free';