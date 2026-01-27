-- 000008_create_subscriptions.up.sql
-- Subscriptions for monetization

-- Subscription plans (reference table)
CREATE TABLE plans (
    id VARCHAR(20) PRIMARY KEY, -- free, pro, agency
    name VARCHAR(50) NOT NULL,
    description TEXT,
    price_monthly DECIMAL(10,2) NOT NULL DEFAULT 0,
    price_yearly DECIMAL(10,2),
    
    -- Limits
    max_photos INTEGER NOT NULL DEFAULT 3,
    max_responses_month INTEGER NOT NULL DEFAULT 5, -- -1 = unlimited
    can_chat BOOLEAN DEFAULT FALSE,
    can_see_viewers BOOLEAN DEFAULT FALSE,
    priority_search BOOLEAN DEFAULT FALSE,
    max_team_members INTEGER DEFAULT 0,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Default plans
INSERT INTO plans (id, name, description, price_monthly, price_yearly, max_photos, max_responses_month, can_chat, can_see_viewers, priority_search, max_team_members) VALUES
('free', 'Free', 'Базовый бесплатный план', 0, NULL, 3, 5, false, false, false, 0),
('pro', 'Pro', 'Профессиональный план для моделей', 3990, 39900, 100, -1, true, true, true, 0),
('agency', 'Agency', 'План для модельных агентств', 14990, 149900, 100, -1, true, true, true, 5);

-- User subscriptions
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id VARCHAR(20) NOT NULL REFERENCES plans(id),
    
    -- Period
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active', -- active, cancelled, expired, pending
    cancelled_at TIMESTAMPTZ,
    cancel_reason TEXT,
    
    -- Billing
    billing_period VARCHAR(10) DEFAULT 'monthly', -- monthly, yearly
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Only one active subscription per user
CREATE UNIQUE INDEX idx_subscriptions_user_active 
    ON subscriptions(user_id) 
    WHERE status = 'active';

CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_expires ON subscriptions(expires_at) WHERE status = 'active';

-- Comments
COMMENT ON TABLE plans IS 'Subscription plans with limits and pricing';
COMMENT ON TABLE subscriptions IS 'User subscription records';
COMMENT ON COLUMN plans.max_responses_month IS '-1 means unlimited';
