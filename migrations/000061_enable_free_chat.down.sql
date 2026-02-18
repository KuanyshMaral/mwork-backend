-- 000061_enable_free_chat.down.sql
-- Disable chat for free plan users

UPDATE plans
SET can_chat = FALSE
WHERE id = 'free';
