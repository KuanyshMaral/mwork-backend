-- User notification preferences and device tokens
-- Migration: 000021_notification_preferences.up.sql

-- Notification preferences per user
CREATE TABLE IF NOT EXISTS user_notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Global channel toggles
    email_enabled BOOLEAN DEFAULT true,
    push_enabled BOOLEAN DEFAULT true,
    in_app_enabled BOOLEAN DEFAULT true,
    
    -- Per-type settings (JSON: {"in_app": true, "email": true, "push": false})
    new_response_channels JSONB DEFAULT '{"in_app": true, "email": true, "push": true}'::jsonb,
    response_accepted_channels JSONB DEFAULT '{"in_app": true, "email": true, "push": true}'::jsonb,
    response_rejected_channels JSONB DEFAULT '{"in_app": true, "email": true, "push": false}'::jsonb,
    new_message_channels JSONB DEFAULT '{"in_app": true, "email": false, "push": true}'::jsonb,
    profile_viewed_channels JSONB DEFAULT '{"in_app": true, "email": false, "push": false}'::jsonb,
    casting_expiring_channels JSONB DEFAULT '{"in_app": true, "email": true, "push": false}'::jsonb,
    
    -- Email preferences
    digest_enabled BOOLEAN DEFAULT true,
    digest_frequency VARCHAR(20) DEFAULT 'weekly', -- 'daily', 'weekly', 'never'
    
    -- Quiet hours (don't send push during these hours)
    quiet_hours_start TIME DEFAULT NULL,
    quiet_hours_end TIME DEFAULT NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT uq_user_notification_preferences UNIQUE(user_id)
);

-- Device tokens for push notifications
CREATE TABLE IF NOT EXISTS device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    token VARCHAR(500) NOT NULL,
    platform VARCHAR(20) NOT NULL, -- 'web', 'android', 'ios'
    device_name VARCHAR(100),
    
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT uq_device_token UNIQUE(token)
);

-- Notification groups for batching
CREATE TABLE IF NOT EXISTS notification_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    type VARCHAR(50) NOT NULL, -- 'new_response', 'new_message', etc.
    count INT DEFAULT 1,
    
    -- Summary data
    summary_data JSONB DEFAULT '{}'::jsonb,
    
    -- First and last notification in group
    first_notification_id UUID REFERENCES notifications(id),
    last_notification_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Whether summary was sent
    summary_sent BOOLEAN DEFAULT false,
    summary_sent_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_notification_prefs_user ON user_notification_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_device_tokens_user ON device_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_device_tokens_active ON device_tokens(user_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_notification_groups_user ON notification_groups(user_id, type);
CREATE INDEX IF NOT EXISTS idx_notification_groups_pending ON notification_groups(summary_sent, last_notification_at) 
    WHERE summary_sent = false;

-- Partial unique index: one active group per user per type
CREATE UNIQUE INDEX IF NOT EXISTS uq_notification_group_active 
    ON notification_groups(user_id, type) 
    WHERE summary_sent = false;

-- Add retention cleanup: notifications older than 90 days
-- This will be run by a scheduled job
COMMENT ON TABLE notifications IS 'Notifications are retained for 90 days, then deleted by cleanup job';

