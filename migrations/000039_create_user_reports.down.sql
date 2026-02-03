-- 000039_create_user_reports.down.sql

DROP INDEX IF EXISTS idx_user_reports_reporter;
DROP INDEX IF EXISTS idx_user_reports_reported_user;
DROP INDEX IF EXISTS idx_user_reports_status_created;
DROP TABLE IF EXISTS user_reports;
DROP TYPE IF EXISTS report_status;
DROP TYPE IF EXISTS report_reason;
