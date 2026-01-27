-- Migration: Add verification fields to users and create organizations table
-- Purpose: Support two-tier registration (instant for models, verified for employers)

-- Add organization type enum
CREATE TYPE org_type AS ENUM ('ip', 'too', 'ao', 'agency', 'other');

-- Add verification status enum
CREATE TYPE verification_status AS ENUM ('none', 'pending', 'in_review', 'verified', 'rejected');

-- Create organizations table
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Legal information
    legal_name VARCHAR(255) NOT NULL,
    brand_name VARCHAR(255),
    bin_iin VARCHAR(12) NOT NULL UNIQUE,  -- БИН/ИИН (12 digits)
    org_type org_type NOT NULL DEFAULT 'too',
    
    -- Address
    legal_address TEXT,
    actual_address TEXT,
    city VARCHAR(100),
    
    -- Contacts
    phone VARCHAR(20),
    email VARCHAR(255),
    website VARCHAR(255),
    
    -- Documents (R2 URLs)
    registration_doc_url TEXT,
    license_doc_url TEXT,
    additional_docs JSONB DEFAULT '[]',
    
    -- Verification
    verification_status verification_status DEFAULT 'pending',
    verification_notes TEXT,
    rejection_reason TEXT,
    verified_at TIMESTAMP,
    verified_by UUID,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Add organization reference to users
ALTER TABLE users ADD COLUMN organization_id UUID REFERENCES organizations(id);
ALTER TABLE users ADD COLUMN user_verification_status verification_status DEFAULT 'none';

-- Update existing employer users to require verification
UPDATE users SET user_verification_status = 'pending' WHERE role = 'employer';
UPDATE users SET user_verification_status = 'verified' WHERE role = 'model';

-- Indexes
CREATE INDEX idx_organizations_bin ON organizations(bin_iin);
CREATE INDEX idx_organizations_status ON organizations(verification_status);
CREATE INDEX idx_organizations_created ON organizations(created_at DESC);
CREATE INDEX idx_users_org ON users(organization_id) WHERE organization_id IS NOT NULL;
CREATE INDEX idx_users_verification ON users(user_verification_status);

-- Comments
COMMENT ON TABLE organizations IS 'Legal entities (companies) that can post castings';
COMMENT ON COLUMN organizations.bin_iin IS 'Business Identification Number (12 digits)';
COMMENT ON COLUMN users.user_verification_status IS 'Verification status for employer accounts';
