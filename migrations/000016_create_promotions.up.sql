-- Migration: Create profile promotions system
-- Purpose: Allow models to promote their profiles for visibility

CREATE TYPE promotion_status AS ENUM ('draft', 'pending_payment', 'active', 'paused', 'completed', 'cancelled');

CREATE TABLE profile_promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- Content
    title VARCHAR(255) NOT NULL,
    description TEXT,
    photo_url TEXT,
    specialization VARCHAR(100),
    
    -- Targeting
    target_audience VARCHAR(50) DEFAULT 'employers',
    target_cities TEXT[] DEFAULT '{}',
    
    -- Budget & Duration
    budget_amount BIGINT NOT NULL,
    daily_budget BIGINT,
    duration_days INT NOT NULL DEFAULT 7,
    
    -- Schedule
    status promotion_status DEFAULT 'draft',
    starts_at TIMESTAMP,
    ends_at TIMESTAMP,
    
    -- Analytics (denormalized for fast access)
    impressions INT DEFAULT 0,
    clicks INT DEFAULT 0,
    responses INT DEFAULT 0,
    spent_amount BIGINT DEFAULT 0,
    
    -- Payment reference
    payment_id UUID REFERENCES payments(id),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Daily statistics for promotions
CREATE TABLE promotion_daily_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promotion_id UUID NOT NULL REFERENCES profile_promotions(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    impressions INT DEFAULT 0,
    clicks INT DEFAULT 0,
    responses INT DEFAULT 0,
    spent BIGINT DEFAULT 0,
    
    UNIQUE(promotion_id, date)
);

-- Indexes
CREATE INDEX idx_promotions_profile ON profile_promotions(profile_id);
CREATE INDEX idx_promotions_status ON profile_promotions(status);
CREATE INDEX idx_promotions_active ON profile_promotions(status, starts_at, ends_at) 
    WHERE status = 'active';
CREATE INDEX idx_promo_stats_promo ON promotion_daily_stats(promotion_id);
CREATE INDEX idx_promo_stats_date ON promotion_daily_stats(date);

-- Comments
COMMENT ON TABLE profile_promotions IS 'Profile advertising campaigns';
COMMENT ON COLUMN profile_promotions.target_audience IS 'employers, agencies, or all';
COMMENT ON TABLE promotion_daily_stats IS 'Daily breakdown of promotion performance';
