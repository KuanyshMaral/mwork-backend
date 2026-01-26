-- Migration: Create reviews system
-- Purpose: Allow models to receive reviews from employers

CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    casting_id UUID REFERENCES castings(id) ON DELETE SET NULL,
    
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    
    is_verified BOOLEAN DEFAULT false,
    is_public BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(profile_id, reviewer_id, casting_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_reviews_profile ON reviews(profile_id);
CREATE INDEX IF NOT EXISTS idx_reviews_rating ON reviews(rating);
CREATE INDEX IF NOT EXISTS idx_reviews_created ON reviews(created_at DESC);

-- Update profile rating trigger
CREATE OR REPLACE FUNCTION update_profile_rating()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE profiles 
    SET rating = (
        SELECT COALESCE(AVG(rating)::numeric(3,2), 0) 
        FROM reviews 
        WHERE profile_id = COALESCE(NEW.profile_id, OLD.profile_id) 
        AND is_public = true
    )
    WHERE id = COALESCE(NEW.profile_id, OLD.profile_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_profile_rating ON reviews;
CREATE TRIGGER trigger_update_profile_rating
AFTER INSERT OR UPDATE OR DELETE ON reviews
FOR EACH ROW EXECUTE FUNCTION update_profile_rating();

COMMENT ON TABLE reviews IS 'Model reviews from employers after completed jobs';
