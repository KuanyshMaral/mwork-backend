-- Migration: Create employer leads table for lead capture
-- Purpose: Capture potential employer contacts before verification

-- Lead status enum
CREATE TYPE lead_status AS ENUM ('new', 'contacted', 'qualified', 'converted', 'rejected', 'lost');

CREATE TABLE employer_leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Contact person
    contact_name VARCHAR(255) NOT NULL,
    contact_email VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(20) NOT NULL,
    contact_position VARCHAR(100),
    
    -- Company info
    company_name VARCHAR(255) NOT NULL,
    bin_iin VARCHAR(12),
    org_type org_type,
    website VARCHAR(255),
    industry VARCHAR(100),
    employees_count VARCHAR(50),  -- '1-10' | '11-50' | '51-200' | '200+'
    
    -- Application details
    use_case TEXT,
    expected_castings_per_month INT,
    how_found_us VARCHAR(100),
    
    -- Lead management
    status lead_status DEFAULT 'new',
    priority INT DEFAULT 0,  -- 0=normal, 1=high, 2=urgent
    assigned_to UUID REFERENCES admin_users(id),
    notes TEXT,
    
    -- Follow-up
    last_contacted_at TIMESTAMP,
    next_follow_up_at TIMESTAMP,
    follow_up_count INT DEFAULT 0,
    
    -- Conversion
    converted_at TIMESTAMP,
    converted_user_id UUID REFERENCES users(id),
    converted_org_id UUID REFERENCES organizations(id),
    rejection_reason TEXT,
    
    -- UTM tracking
    source VARCHAR(50),  -- 'website' | 'referral' | 'ad' | 'social'
    utm_source VARCHAR(100),
    utm_medium VARCHAR(100),
    utm_campaign VARCHAR(100),
    referrer_url TEXT,
    
    -- Metadata
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_leads_status ON employer_leads(status);
CREATE INDEX idx_leads_email ON employer_leads(contact_email);
CREATE INDEX idx_leads_priority ON employer_leads(priority DESC, created_at DESC);
CREATE INDEX idx_leads_assigned ON employer_leads(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_leads_follow_up ON employer_leads(next_follow_up_at) WHERE next_follow_up_at IS NOT NULL;
CREATE INDEX idx_leads_created ON employer_leads(created_at DESC);

-- Comments
COMMENT ON TABLE employer_leads IS 'Lead capture for potential employer accounts';
COMMENT ON COLUMN employer_leads.use_case IS 'What they want to use the platform for';
COMMENT ON COLUMN employer_leads.priority IS '0=normal, 1=high, 2=urgent';
