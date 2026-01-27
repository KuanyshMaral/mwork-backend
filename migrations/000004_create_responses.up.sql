-- migrations/000004_create_responses.up.sql
-- Responses table for casting applications

CREATE TABLE responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    casting_id UUID NOT NULL REFERENCES castings(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,

    status VARCHAR(20) DEFAULT 'pending' 
        CHECK (status IN ('pending', 'viewed', 'accepted', 'rejected')),
    cover_letter TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Prevent duplicate applications
    CONSTRAINT unique_application UNIQUE(casting_id, profile_id)
);

-- Indexes
CREATE INDEX idx_responses_casting ON responses(casting_id, status);
CREATE INDEX idx_responses_profile ON responses(profile_id);
CREATE INDEX idx_responses_created_at ON responses(created_at DESC);
CREATE INDEX idx_responses_status ON responses(status);

-- Trigger for updated_at
CREATE TRIGGER responses_updated_at
    BEFORE UPDATE ON responses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to update casting response_count
CREATE OR REPLACE FUNCTION update_casting_response_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE castings 
        SET response_count = response_count + 1 
        WHERE id = NEW.casting_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE castings 
        SET response_count = response_count - 1 
        WHERE id = OLD.casting_id;
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update response_count
CREATE TRIGGER responses_count_trigger
    AFTER INSERT OR DELETE ON responses
    FOR EACH ROW
    EXECUTE FUNCTION update_casting_response_count();

-- Comments
COMMENT ON TABLE responses IS 'Applications from models to castings';
COMMENT ON COLUMN responses.status IS 'Application status: pending, viewed, accepted, or rejected';
