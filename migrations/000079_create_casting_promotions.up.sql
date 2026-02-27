-- Migration: Create casting promotions system
-- Purpose: Allow employers to promote their castings for higher visibility

CREATE TABLE casting_promotions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    casting_id UUID NOT NULL REFERENCES castings(id) ON DELETE CASCADE,
    employer_id UUID NOT NULL REFERENCES employer_profiles(id) ON DELETE CASCADE,

    -- Optional overrides (if not set, casting's own title/cover is used on the frontend)
    custom_title VARCHAR(255),
    custom_photo_url TEXT,

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

-- Daily statistics for casting promotions
CREATE TABLE casting_promotion_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    promotion_id UUID NOT NULL REFERENCES casting_promotions(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    impressions INT DEFAULT 0,
    clicks INT DEFAULT 0,
    responses INT DEFAULT 0,
    spent BIGINT DEFAULT 0,

    UNIQUE(promotion_id, date)
);

-- Indexes
CREATE INDEX idx_casting_promotions_casting ON casting_promotions(casting_id);
CREATE INDEX idx_casting_promotions_employer ON casting_promotions(employer_id);
CREATE INDEX idx_casting_promotions_status ON casting_promotions(status);
CREATE INDEX idx_casting_promotions_active ON casting_promotions(status, starts_at, ends_at)
    WHERE status = 'active';
CREATE INDEX idx_casting_promo_stats_promo ON casting_promotion_stats(promotion_id);
CREATE INDEX idx_casting_promo_stats_date ON casting_promotion_stats(date);

COMMENT ON TABLE casting_promotions IS 'Casting advertising campaigns by employers';
COMMENT ON TABLE casting_promotion_stats IS 'Daily breakdown of casting promotion performance';
