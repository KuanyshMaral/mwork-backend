-- 000005_create_photos.up.sql
-- Photos table for model portfolios

CREATE TABLE photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- R2 Object info
    key VARCHAR(255) NOT NULL UNIQUE, -- R2 object key
    url VARCHAR(500) NOT NULL,         -- Public CDN URL
    
    -- Original file info
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL,
    
    -- Metadata
    is_avatar BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_photos_profile ON photos(profile_id, sort_order);
CREATE INDEX idx_photos_avatar ON photos(profile_id) WHERE is_avatar = true;

-- Only one avatar per profile constraint
CREATE UNIQUE INDEX idx_photos_unique_avatar ON photos(profile_id) WHERE is_avatar = true;

-- Comments
COMMENT ON TABLE photos IS 'Photo gallery for model portfolios (files stored in R2)';
COMMENT ON COLUMN photos.key IS 'Cloudflare R2 object key (path in bucket)';
COMMENT ON COLUMN photos.url IS 'Public CDN URL for direct access';
