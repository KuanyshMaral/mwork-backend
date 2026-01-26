-- 000007_create_notifications.up.sql
-- Notifications for user alerts

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Notification type and content
    type VARCHAR(50) NOT NULL, -- new_response, response_accepted, response_rejected, new_message, profile_viewed
    title VARCHAR(200) NOT NULL,
    body TEXT,
    
    -- Related entities (for deep linking)
    data JSONB, -- {casting_id, response_id, profile_id, room_id, etc.}
    
    -- Status
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_notifications_user ON notifications(user_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(user_id) WHERE NOT is_read;
CREATE INDEX idx_notifications_type ON notifications(user_id, type);

-- Comments  
COMMENT ON TABLE notifications IS 'User notifications for various events';
COMMENT ON COLUMN notifications.data IS 'JSON with related entity IDs for deep linking';
