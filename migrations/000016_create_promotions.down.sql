-- Rollback: Remove promotions system

DROP TABLE IF EXISTS promotion_daily_stats;
DROP TABLE IF EXISTS profile_promotions;
DROP TYPE IF EXISTS promotion_status;
