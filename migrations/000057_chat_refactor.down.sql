-- 000057_chat_refactor.down.sql
-- Rollback chat refactor migration

-- 1. Re-add participant columns
ALTER TABLE chat_rooms ADD COLUMN participant1_id UUID;
ALTER TABLE chat_rooms ADD COLUMN participant2_id UUID;

-- 2. Migrate data back from chat_room_members (for direct rooms only)
-- First participant (smallest user_id)
UPDATE chat_rooms cr
SET participant1_id = (
    SELECT user_id 
    FROM chat_room_members 
    WHERE room_id = cr.id 
    ORDER BY user_id ASC 
    LIMIT 1
)
WHERE room_type = 'direct';

-- Second participant (largest user_id)
UPDATE chat_rooms cr
SET participant2_id = (
    SELECT user_id 
    FROM chat_room_members 
    WHERE room_id = cr.id 
    ORDER BY user_id DESC 
    LIMIT 1
)
WHERE room_type = 'direct';

-- 3. Re-add foreign key constraints
ALTER TABLE chat_rooms ALTER COLUMN participant1_id SET NOT NULL;
ALTER TABLE chat_rooms ALTER COLUMN participant2_id SET NOT NULL;
ALTER TABLE chat_rooms ADD CONSTRAINT fk_participant1 FOREIGN KEY (participant1_id) REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE chat_rooms ADD CONSTRAINT fk_participant2 FOREIGN KEY (participant2_id) REFERENCES users(id) ON DELETE CASCADE;

-- 4. Re-add unique constraints
ALTER TABLE chat_rooms ADD CONSTRAINT unique_participants CHECK (participant1_id < participant2_id);
ALTER TABLE chat_rooms ADD CONSTRAINT unique_room UNIQUE (participant1_id, participant2_id);

-- 5. Drop new indexes
DROP INDEX IF EXISTS idx_chat_room_members_role;
DROP INDEX IF EXISTS idx_chat_room_members_user;
DROP INDEX IF EXISTS idx_chat_room_members_room;
DROP INDEX IF EXISTS idx_messages_attachments;

-- 6. Drop chat_room_members table
DROP TABLE IF EXISTS chat_room_members;

-- 7. Remove attachment column from messages
ALTER TABLE messages DROP COLUMN IF EXISTS attachment_upload_id;

-- 8. Remove new columns from chat_rooms
ALTER TABLE chat_rooms DROP COLUMN IF EXISTS creator_id;
ALTER TABLE chat_rooms DROP COLUMN IF EXISTS name;
ALTER TABLE chat_rooms DROP CONSTRAINT IF EXISTS check_room_type;
ALTER TABLE chat_rooms DROP COLUMN IF EXISTS room_type;

-- 9. Delete non-direct rooms (cannot be represented in old schema)
DELETE FROM chat_rooms WHERE participant1_id IS NULL OR participant2_id IS NULL;
