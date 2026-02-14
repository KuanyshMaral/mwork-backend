-- 000038_create_user_blocks.up.sql
-- User blocking functionality for chat moderation

CREATE TABLE IF NOT EXISTS user_blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    blocker_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure uniqueness and prevent self-blocking
    CONSTRAINT uniq_block UNIQUE(blocker_user_id, blocked_user_id),
    CONSTRAINT no_self_block CHECK (blocker_user_id <> blocked_user_id)
);

-- Indexes for efficient blocking checks
CREATE INDEX idx_user_blocks_blocker ON user_blocks(blocker_user_id);
CREATE INDEX idx_user_blocks_blocked ON user_blocks(blocked_user_id);

-- Comments
COMMENT ON TABLE user_blocks IS 'User blocking relationships - prevents chat and room creation';
COMMENT ON COLUMN user_blocks.blocker_user_id IS 'User who initiated the block';
COMMENT ON COLUMN user_blocks.blocked_user_id IS 'User who is blocked';
