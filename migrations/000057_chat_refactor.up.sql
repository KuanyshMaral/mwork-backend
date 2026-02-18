-- 000057_chat_refactor.up.sql
-- Force update: 2026-02-19 Correct Version (Chat - Idempotent Fix v2)
-- Refactor chat to support multiple room types (direct, casting, group)

-- 1. Add room_type column to chat_rooms
ALTER TABLE chat_rooms ADD COLUMN IF NOT EXISTS room_type VARCHAR(10) NOT NULL DEFAULT 'direct';

-- Drop check constraint if exists, then re-add
ALTER TABLE chat_rooms DROP CONSTRAINT IF EXISTS check_room_type;
ALTER TABLE chat_rooms ADD CONSTRAINT check_room_type CHECK (room_type IN ('direct', 'casting', 'group'));

-- 2. Add name and creator_id for group chats
ALTER TABLE chat_rooms ADD COLUMN IF NOT EXISTS name VARCHAR(100);
ALTER TABLE chat_rooms ADD COLUMN IF NOT EXISTS creator_id UUID;

-- Handle foreign key for creator_id safely
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chat_rooms_creator_id_fkey') THEN 
        ALTER TABLE chat_rooms ADD CONSTRAINT chat_rooms_creator_id_fkey FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE SET NULL;
    END IF; 
END $$;

-- 3. Create chat_room_members table to replace participant1/participant2
CREATE TABLE IF NOT EXISTS chat_room_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(10) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT check_member_role CHECK (role IN ('admin', 'member')),
    CONSTRAINT unique_room_member UNIQUE (room_id, user_id)
);

-- 4. Migrate existing participants to chat_room_members (Safe Data Migration)
-- Only attempt to select from participant columns if they exist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='chat_rooms' AND column_name='participant1_id') THEN
        INSERT INTO chat_room_members (room_id, user_id, role, joined_at)
        SELECT 
            id as room_id,
            participant1_id as user_id,
            'member' as role,
            created_at as joined_at
        FROM chat_rooms
        WHERE participant1_id IS NOT NULL
        ON CONFLICT DO NOTHING;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='chat_rooms' AND column_name='participant2_id') THEN
        INSERT INTO chat_room_members (room_id, user_id, role, joined_at)
        SELECT 
            id as room_id,
            participant2_id as user_id,
            'member' as role,
            created_at as joined_at
        FROM chat_rooms
        WHERE participant2_id IS NOT NULL
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

-- 5. Add attachment support to messages
ALTER TABLE messages ADD COLUMN IF NOT EXISTS attachment_upload_id UUID;

-- Handle foreign key for attachment_upload_id safely
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'messages_attachment_upload_id_fkey') THEN 
        ALTER TABLE messages ADD CONSTRAINT messages_attachment_upload_id_fkey FOREIGN KEY (attachment_upload_id) REFERENCES uploads(id) ON DELETE SET NULL;
    END IF; 
END $$;

-- 6. Drop old participant columns and constraints
ALTER TABLE chat_rooms DROP CONSTRAINT IF EXISTS unique_participants;
ALTER TABLE chat_rooms DROP CONSTRAINT IF EXISTS unique_room;
ALTER TABLE chat_rooms DROP COLUMN IF EXISTS participant1_id;
ALTER TABLE chat_rooms DROP COLUMN IF EXISTS participant2_id;

-- 7. Create indexes for chat_room_members
CREATE INDEX IF NOT EXISTS idx_chat_room_members_room ON chat_room_members(room_id);
CREATE INDEX IF NOT EXISTS idx_chat_room_members_user ON chat_room_members(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_room_members_role ON chat_room_members(room_id, role) WHERE role = 'admin';

-- 8. Create index for messages with attachments
CREATE INDEX IF NOT EXISTS idx_messages_attachments ON messages(attachment_upload_id) WHERE attachment_upload_id IS NOT NULL;

-- Comments
COMMENT ON TABLE chat_room_members IS 'Members of chat rooms with their roles';
COMMENT ON COLUMN chat_rooms.room_type IS 'Type of room: direct (1-to-1), casting (linked to casting), group (multi-user)';
COMMENT ON COLUMN chat_rooms.creator_id IS 'User who created the room (admin for group chats)';
COMMENT ON COLUMN messages.attachment_upload_id IS 'Reference to uploaded file attachment';
