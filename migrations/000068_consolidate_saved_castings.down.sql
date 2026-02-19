-- Rollback 000067: Recreate saved_castings and restore data from favorites

CREATE TABLE IF NOT EXISTS saved_castings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    casting_id UUID NOT NULL REFERENCES castings(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, casting_id)
);

CREATE INDEX IF NOT EXISTS idx_saved_user ON saved_castings(user_id);
CREATE INDEX IF NOT EXISTS idx_saved_casting ON saved_castings(casting_id);

-- Restore data from favorites
INSERT INTO saved_castings (user_id, casting_id, created_at)
SELECT user_id, entity_id, created_at
FROM favorites
WHERE entity_type = 'casting';
