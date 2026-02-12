-- 000039_create_user_reports.up.sql
-- User reporting and moderation system

CREATE TYPE report_reason AS ENUM ('spam', 'abuse', 'scam', 'nudity', 'other');
CREATE TYPE report_status AS ENUM ('pending', 'reviewing', 'resolved', 'dismissed');

CREATE TABLE IF NOT EXISTS user_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reported_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    room_id UUID NULL REFERENCES chat_rooms(id) ON DELETE SET NULL,
    message_id UUID NULL REFERENCES messages(id) ON DELETE SET NULL,
    
    -- Report details
    reason report_reason NOT NULL,
    description TEXT NULL,
    
    -- Moderation status
    status report_status NOT NULL DEFAULT 'pending',
    admin_notes TEXT NULL,
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ NULL
);

-- Indexes for efficient querying
CREATE INDEX idx_user_reports_status_created ON user_reports(status, created_at DESC);
CREATE INDEX idx_user_reports_reported_user ON user_reports(reported_user_id);
CREATE INDEX idx_user_reports_reporter ON user_reports(reporter_user_id);

-- Comments
COMMENT ON TABLE user_reports IS 'User-generated reports for moderation';
COMMENT ON COLUMN user_reports.reporter_user_id IS 'User who submitted the report';
COMMENT ON COLUMN user_reports.reported_user_id IS 'User being reported';
COMMENT ON COLUMN user_reports.room_id IS 'Optional: chat room context';
COMMENT ON COLUMN user_reports.message_id IS 'Optional: specific message being reported';
COMMENT ON COLUMN user_reports.reason IS 'Category of the report';
COMMENT ON COLUMN user_reports.status IS 'Current moderation status';
