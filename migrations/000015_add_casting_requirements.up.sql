-- Migration: Add casting requirements and saved castings
-- Purpose: Support detailed casting requirements and favorites functionality

-- Casting requirements for model matching
ALTER TABLE castings ADD COLUMN IF NOT EXISTS required_gender VARCHAR(20);
ALTER TABLE castings ADD COLUMN IF NOT EXISTS min_age INT;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS max_age INT;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS min_height INT;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS max_height INT;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS min_weight INT;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS max_weight INT;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS required_experience VARCHAR(50);
ALTER TABLE castings ADD COLUMN IF NOT EXISTS required_languages TEXT[] DEFAULT '{}';
ALTER TABLE castings ADD COLUMN IF NOT EXISTS clothing_sizes TEXT[] DEFAULT '{}';
ALTER TABLE castings ADD COLUMN IF NOT EXISTS shoe_sizes TEXT[] DEFAULT '{}';

-- Work details
ALTER TABLE castings ADD COLUMN IF NOT EXISTS work_type VARCHAR(50) DEFAULT 'one_time';
ALTER TABLE castings ADD COLUMN IF NOT EXISTS event_datetime TIMESTAMP;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS event_location TEXT;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS deadline_at TIMESTAMP;
ALTER TABLE castings ADD COLUMN IF NOT EXISTS is_urgent BOOLEAN DEFAULT false;

-- Views counter
ALTER TABLE castings ADD COLUMN IF NOT EXISTS views_count INT DEFAULT 0;

-- Indexes for filtering
CREATE INDEX IF NOT EXISTS idx_castings_gender ON castings(required_gender) WHERE required_gender IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_castings_age ON castings(min_age, max_age);
CREATE INDEX IF NOT EXISTS idx_castings_height ON castings(min_height, max_height);
CREATE INDEX IF NOT EXISTS idx_castings_urgent ON castings(is_urgent) WHERE is_urgent = true;
CREATE INDEX IF NOT EXISTS idx_castings_deadline ON castings(deadline_at) WHERE deadline_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_castings_event ON castings(event_datetime) WHERE event_datetime IS NOT NULL;

-- Saved castings (favorites)
CREATE TABLE IF NOT EXISTS saved_castings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    casting_id UUID NOT NULL REFERENCES castings(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(user_id, casting_id)
);

CREATE INDEX IF NOT EXISTS idx_saved_user ON saved_castings(user_id);
CREATE INDEX IF NOT EXISTS idx_saved_casting ON saved_castings(casting_id);

-- Comments
COMMENT ON COLUMN castings.work_type IS 'one_time, contract, or permanent';
COMMENT ON COLUMN castings.required_experience IS 'none, beginner, medium, professional';
COMMENT ON TABLE saved_castings IS 'User favorites/bookmarks for castings';
