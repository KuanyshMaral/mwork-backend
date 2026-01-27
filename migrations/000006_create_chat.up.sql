-- 000006_create_chat.up.sql
-- Chat rooms and messages for real-time communication

-- Chat rooms (1-to-1 between model and employer)
CREATE TABLE chat_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Participants (always 2: model + employer)
    participant1_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    participant2_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Optional link to casting
    casting_id UUID REFERENCES castings(id) ON DELETE SET NULL,
    
    -- Last message preview
    last_message_at TIMESTAMPTZ,
    last_message_preview VARCHAR(100),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure unique pair (order doesn't matter)
    CONSTRAINT unique_participants CHECK (participant1_id < participant2_id),
    CONSTRAINT unique_room UNIQUE (participant1_id, participant2_id)
);

-- Messages
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Content
    content TEXT NOT NULL,
    message_type VARCHAR(20) DEFAULT 'text', -- text, image, system
    
    -- Read status
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMPTZ,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Soft delete
    deleted_at TIMESTAMPTZ
);

-- Indexes for chat_rooms
CREATE INDEX idx_chat_rooms_participant1 ON chat_rooms(participant1_id);
CREATE INDEX idx_chat_rooms_participant2 ON chat_rooms(participant2_id);
CREATE INDEX idx_chat_rooms_last_message ON chat_rooms(last_message_at DESC NULLS LAST);

-- Indexes for messages
CREATE INDEX idx_messages_room ON messages(room_id, created_at DESC);
CREATE INDEX idx_messages_unread ON messages(room_id, sender_id) WHERE NOT is_read AND deleted_at IS NULL;

-- Comments
COMMENT ON TABLE chat_rooms IS 'Direct message rooms between two users';
COMMENT ON TABLE messages IS 'Chat messages within rooms';
COMMENT ON CONSTRAINT unique_participants ON chat_rooms IS 'Ensures participant1_id < participant2_id for consistent ordering';
