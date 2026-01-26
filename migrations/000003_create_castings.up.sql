-- migrations/000003_create_castings.up.sql
-- Castings table for job postings

CREATE TABLE castings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Basic info
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    
    -- Location
    city VARCHAR(100) NOT NULL,
    address VARCHAR(500),
    
    -- Payment
    pay_min DECIMAL(10,2) CHECK (pay_min IS NULL OR pay_min >= 0),
    pay_max DECIMAL(10,2) CHECK (pay_max IS NULL OR pay_max >= 0),
    pay_type VARCHAR(20) DEFAULT 'negotiable'
        CHECK (pay_type IN ('fixed', 'hourly', 'negotiable', 'free')),
    
    -- Dates
    date_from TIMESTAMPTZ,
    date_to TIMESTAMPTZ,
    
    -- Requirements (flexible JSONB structure)
    requirements JSONB DEFAULT '{}',
    /*
    Example structure:
    {
      "gender": "female",
      "age_min": 18,
      "age_max": 25,
      "height_min": 170,
      "height_max": 180,
      "experience_required": false,
      "languages": ["russian", "english"]
    }
    */
    
    -- Status and promotion
    status VARCHAR(20) DEFAULT 'active' 
        CHECK (status IN ('draft', 'active', 'closed', 'deleted')),
    is_promoted BOOLEAN DEFAULT FALSE,
    
    -- Stats
    view_count INTEGER DEFAULT 0,
    response_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_pay_range CHECK (pay_min IS NULL OR pay_max IS NULL OR pay_min <= pay_max),
    CONSTRAINT valid_date_range CHECK (date_from IS NULL OR date_to IS NULL OR date_from <= date_to)
);

-- Indexes for common queries
CREATE INDEX idx_castings_creator ON castings(creator_id);
CREATE INDEX idx_castings_city_status ON castings(city, status) WHERE status = 'active';
CREATE INDEX idx_castings_created_at ON castings(created_at DESC);
CREATE INDEX idx_castings_pay ON castings(pay_max DESC NULLS LAST) WHERE status = 'active';
CREATE INDEX idx_castings_date ON castings(date_from) WHERE status = 'active';

-- Full-text search index
CREATE INDEX idx_castings_fts ON castings 
    USING GIN(to_tsvector('russian', coalesce(title, '') || ' ' || coalesce(description, '')));

-- Trigger for updated_at
CREATE TRIGGER castings_updated_at
    BEFORE UPDATE ON castings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE castings IS 'Job postings/castings created by employers';
COMMENT ON COLUMN castings.requirements IS 'Flexible JSON structure for model requirements';
COMMENT ON COLUMN castings.pay_type IS 'Payment type: fixed, hourly, negotiable, or free';
