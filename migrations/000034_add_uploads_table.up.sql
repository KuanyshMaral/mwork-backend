-- 000034_add_uploads_table.up.sql
-- Create uploads table for tracking 2-phase upload flow (init -> confirm)

CREATE TABLE IF NOT EXISTS uploads (
                                       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                       user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                       temp_path VARCHAR(500),
                                       final_path VARCHAR(500),
                                       status VARCHAR(20) NOT NULL DEFAULT 'pending'
                                           CHECK (status IN ('pending', 'confirmed', 'failed')),
                                       created_at TIMESTAMP NOT NULL DEFAULT NOW(),
                                       confirmed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_uploads_user_id
    ON uploads(user_id);

CREATE INDEX IF NOT EXISTS idx_uploads_status
    ON uploads(status);

COMMENT ON TABLE uploads
    IS 'Two-phase upload tracking (init → confirm)';
