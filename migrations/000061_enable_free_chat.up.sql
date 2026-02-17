-- 000061_enable_free_chat.up.sql
-- Enable chat for free plan users

UPDATE plans
SET can_chat = TRUE
WHERE id = 'free';
