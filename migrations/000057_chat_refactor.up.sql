-- 000057_chat_refactor.up.sql
-- Refactor chat to support multiple room types (direct, casting, group)

-- 1. Add room_type column to chat_rooms
ALTER TABLE chat_rooms ADD COLUMN room_type VARCHAR(10) NOT NULL DEFAULT 'direct';
ALTER TABLE chat_rooms ADD CONSTRAINT check_room_type CHECK (room_type IN ('direct', 'casting', 'group'));

-- 2. Add name and creator_id for group chats
ALTER TABLE chat_rooms ADD COLUMN name VARCHAR(100);
ALTER TABLE chat_rooms ADD COLUMN creator_id UUID REFERENCES users(id) ON DELETE SET NULL;

-- 3. Create chat_room_members table to replace participant1/participant2
CREATE TABLE chat_room_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(10) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT check_member_role CHECK (role IN ('admin', 'member')),
    CONSTRAINT unique_room_member UNIQUE (room_id, user_id)
);

-- 4. Migrate existing participants to chat_room_members
INSERT INTO chat_room_members (room_id, user_id, role, joined_at)
SELECT 
    id as room_id,
    participant1_id as user_id,
    'member' as role,
    created_at as joined_at
FROM chat_rooms
WHERE participant1_id IS NOT NULL;

INSERT INTO chat_room_members (room_id, user_id, role, joined_at)
SELECT 
    id as room_id,
    participant2_id as user_id,
    'member' as role,
    created_at as joined_at
FROM chat_rooms
WHERE participant2_id IS NOT NULL;

-- 5. Add attachment support to messages
ALTER TABLE messages ADD COLUMN attachment_upload_id UUID REFERENCES uploads(id) ON DELETE SET NULL;

-- 6. Drop old participant columns and constraints
ALTER TABLE chat_rooms DROP CONSTRAINT IF EXISTS unique_participants;
ALTER TABLE chat_rooms DROP CONSTRAINT IF EXISTS unique_room;
ALTER TABLE chat_rooms DROP COLUMN participant1_id;
ALTER TABLE chat_rooms DROP COLUMN participant2_id;

-- 7. Create indexes for chat_room_members
CREATE INDEX idx_chat_room_members_room ON chat_room_members(room_id);
CREATE INDEX idx_chat_room_members_user ON chat_room_members(user_id);
CREATE INDEX idx_chat_room_members_role ON chat_room_members(room_id, role) WHERE role = 'admin';

-- 8. Create index for messages with attachments
CREATE INDEX idx_messages_attachments ON messages(attachment_upload_id) WHERE attachment_upload_id IS NOT NULL;

-- Comments
COMMENT ON TABLE chat_room_members IS 'Members of chat rooms with their roles';
COMMENT ON COLUMN chat_rooms.room_type IS 'Type of room: direct (1-to-1), casting (linked to casting), group (multi-user)';
COMMENT ON COLUMN chat_rooms.creator_id IS 'User who created the room (admin for group chats)';
COMMENT ON COLUMN messages.attachment_upload_id IS 'Reference to uploaded file attachment';
