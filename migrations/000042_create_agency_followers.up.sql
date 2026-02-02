-- Create agency_followers table
CREATE TABLE IF NOT EXISTS agency_followers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    follower_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_organization_follower UNIQUE (organization_id, follower_user_id)
);

-- Create indexes
CREATE INDEX idx_agency_followers_org_id ON agency_followers(organization_id);
CREATE INDEX idx_agency_followers_user_id ON agency_followers(follower_user_id);
CREATE INDEX idx_agency_followers_created_at ON agency_followers(created_at DESC);
