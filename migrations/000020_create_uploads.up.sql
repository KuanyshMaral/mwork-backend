-- migrations/000010_create_uploads.up.sql
-- File uploads with 2-phase lifecycle (staged -> committed)

CREATE TABLE uploads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Category and status
    category VARCHAR(20) NOT NULL CHECK (category IN ('avatar', 'photo', 'document')),
    status VARCHAR(20) NOT NULL DEFAULT 'staged' 
        CHECK (status IN ('staged', 'committed', 'failed', 'deleted')),
    
    -- File metadata
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    size BIGINT NOT NULL CHECK (size > 0),
    
    -- Storage keys
    staging_key VARCHAR(500),
    permanent_key VARCHAR(500),
    permanent_url VARCHAR(1000),
    
    -- Image dimensions (for images)
    width INTEGER DEFAULT 0,
    height INTEGER DEFAULT 0,
    
    -- Error tracking
    error_message TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    committed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    
    -- Constraints
    CONSTRAINT staging_or_permanent CHECK (
        (status = 'staged' AND staging_key IS NOT NULL) OR
        (status = 'committed' AND permanent_key IS NOT NULL) OR
        (status IN ('failed', 'deleted'))
    )
);

-- Indexes
CREATE INDEX idx_uploads_user ON uploads(user_id);
CREATE INDEX idx_uploads_user_category ON uploads(user_id, category) WHERE status = 'committed';
CREATE INDEX idx_uploads_status ON uploads(status);
CREATE INDEX idx_uploads_expires ON uploads(expires_at) WHERE status = 'staged';

-- Comments
COMMENT ON TABLE uploads IS 'File uploads with 2-phase lifecycle';
COMMENT ON COLUMN uploads.staging_key IS 'Key in staging (temporary) storage';
COMMENT ON COLUMN uploads.permanent_key IS 'Key in permanent (cloud) storage';
COMMENT ON COLUMN uploads.expires_at IS 'Expiration time for staged files';
