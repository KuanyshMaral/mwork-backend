-- 000038_create_user_blocks.down.sql

DROP INDEX IF EXISTS idx_user_blocks_blocked;
DROP INDEX IF EXISTS idx_user_blocks_blocker;
DROP TABLE IF EXISTS user_blocks;
