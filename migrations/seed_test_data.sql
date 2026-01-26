-- seed_test_data.sql (CORRECTED for actual schema)
-- Run: psql -U postgres -d mwork -f migrations/seed_test_data.sql
-- Or: migrate -path migrations -database "postgres://..." -source file://migrations/seed_test_data.sql force 1

-- ============================================
-- 0. PLANS are already seeded in migration 000008
-- Just update if needed
-- ============================================

UPDATE plans SET 
    price_monthly = 0,
    max_photos = 5,
    max_responses_month = 20
WHERE id = 'free';

UPDATE plans SET 
    price_monthly = 4990,
    max_photos = 50,
    max_responses_month = -1
WHERE id = 'pro';

-- ============================================
-- 1. USERS (password: password123)
-- Hash: $2a$12$... (bcrypt)
-- ============================================

-- Model 1
INSERT INTO users (id, email, password_hash, role, email_verified, is_banned) VALUES
('11111111-1111-1111-1111-111111111111', 
 'test1@test.com', 
 '$2a$12$/RLssaN.rEXydJMSl9ImvOnA9s5tL0Ia54sNlVt4H0nj3x6XnilA2', 
 'model', 
 true,
 false)
ON CONFLICT (email) DO UPDATE SET 
    password_hash = EXCLUDED.password_hash,
    email_verified = EXCLUDED.email_verified,
    is_banned = EXCLUDED.is_banned;

-- Model 2
INSERT INTO users (id, email, password_hash, role, email_verified, is_banned) VALUES
('22222222-2222-2222-2222-222222222222', 
 'model2@test.com', 
 '$2a$12$/RLssaN.rEXydJMSl9ImvOnA9s5tL0Ia54sNlVt4H0nj3x6XnilA2', 
 'model', 
 true, 
 false)
ON CONFLICT (email) DO UPDATE SET 
    password_hash = EXCLUDED.password_hash,
    email_verified = EXCLUDED.email_verified,
    is_banned = EXCLUDED.is_banned;

-- Employer 1
INSERT INTO users (id, email, password_hash, role, email_verified, is_banned) VALUES
('33333333-3333-3333-3333-333333333333', 
 'employer1@test.com', 
 '$2a$12$/RLssaN.rEXydJMSl9ImvOnA9s5tL0Ia54sNlVt4H0nj3x6XnilA2', 
 'employer', 
 true, 
 false)
ON CONFLICT (email) DO UPDATE SET 
    password_hash = EXCLUDED.password_hash,
    email_verified = EXCLUDED.email_verified,
    is_banned = EXCLUDED.is_banned;

-- Employer 2
INSERT INTO users (id, email, password_hash, role, email_verified, is_banned) VALUES
('44444444-4444-4444-4444-444444444444', 
 'employer2@test.com', 
 '$2a$12$/RLssaN.rEXydJMSl9ImvOnA9s5tL0Ia54sNlVt4H0nj3x6XnilA2', 
 'employer', 
 true, 
 false)
ON CONFLICT (email) DO UPDATE SET 
    password_hash = EXCLUDED.password_hash,
    email_verified = EXCLUDED.email_verified,
    is_banned = EXCLUDED.is_banned;

-- ============================================
-- 2. PROFILES (single table with type)
-- ============================================

-- Model Profile 1 (Anna)
INSERT INTO profiles (id, user_id, type, first_name, last_name, bio, age, height_cm, weight_kg, gender, 
                      experience_years, hourly_rate, city, languages, is_public) 
VALUES
('aaaa1111-1111-1111-1111-111111111111',
 '11111111-1111-1111-1111-111111111111',
 'model',
 'Anna', 'Ivanova',
 'Professional model with 3 years experience. Worked with Vogue, ELLE.',
 23, 175, 55, 'female',
 3, 25000.00, 'Almaty',
 '["russian", "english"]'::jsonb,
 true)
ON CONFLICT (user_id) DO UPDATE SET 
    first_name = EXCLUDED.first_name,
    bio = EXCLUDED.bio,
    is_public = EXCLUDED.is_public;

-- Model Profile 2 (Marat)
INSERT INTO profiles (id, user_id, type, first_name, last_name, bio, age, height_cm, weight_kg, gender,
                      experience_years, hourly_rate, city, languages, is_public)
VALUES
('aaaa2222-2222-2222-2222-222222222222',
 '22222222-2222-2222-2222-222222222222',
 'model',
 'Marat', 'Nurlanov',
 'Male model. Experience in fashion shows and commercial shoots.',
 27, 185, 78, 'male',
 2, 30000.00, 'Astana',
 '["russian", "kazakh", "english"]'::jsonb,
 true)
ON CONFLICT (user_id) DO UPDATE SET 
    first_name = EXCLUDED.first_name,
    bio = EXCLUDED.bio,
    is_public = EXCLUDED.is_public;

-- Employer Profile 1
INSERT INTO profiles (id, user_id, type, first_name, company_name, bio, city, is_public)
VALUES
('bbbb3333-3333-3333-3333-333333333333',
 '33333333-3333-3333-3333-333333333333',
 'employer',
 'Alexey', 'Fashion Elite Studio',
 'Professional photo studio for fashion and commercial photography.',
 'Almaty',
 true)
ON CONFLICT (user_id) DO UPDATE SET 
    company_name = EXCLUDED.company_name,
    bio = EXCLUDED.bio;

-- Employer Profile 2
INSERT INTO profiles (id, user_id, type, first_name, company_name, bio, city, is_public)
VALUES
('bbbb4444-4444-4444-4444-444444444444',
 '44444444-4444-4444-4444-444444444444',
 'employer',
 'Dinara', 'ELITE Models Kazakhstan',
 'Leading modeling agency in Kazakhstan.',
 'Almaty',
 true)
ON CONFLICT (user_id) DO UPDATE SET 
    company_name = EXCLUDED.company_name,
    bio = EXCLUDED.bio;

-- ============================================
-- 3. CASTINGS (uses creator_id, not employer_id!)
-- ============================================

-- Casting 1: Vogue photoshoot (Employer 1)
INSERT INTO castings (id, creator_id, title, description, city, 
                      pay_min, pay_max, pay_type, date_from,
                      requirements, status)
VALUES
('cccc1111-1111-1111-1111-111111111111',
 '33333333-3333-3333-3333-333333333333',
 'Elite photoshoot for Vogue',
 'Looking for professional models for a new collection shoot. Experience required. Professional studio with top photographer.',
 'Almaty',
 50000, 100000, 'negotiable',
 NOW() + interval '5 days',
 '{"gender": "female", "age_min": 18, "age_max": 28, "height_min": 170}'::jsonb,
 'active')
ON CONFLICT (id) DO NOTHING;

-- Casting 2: Samsung campaign (Employer 2)
INSERT INTO castings (id, creator_id, title, description, city,
                      pay_min, pay_max, pay_type, date_from,
                      requirements, status)
VALUES
('cccc2222-2222-2222-2222-222222222222',
 '44444444-4444-4444-4444-444444444444',
 'Samsung advertising campaign',
 'Shooting for the new Galaxy smartphone advertising campaign. Looking for young and energetic models.',
 'Almaty',
 75000, 75000, 'fixed',
 NOW() + interval '10 days',
 '{"age_min": 20, "age_max": 35}'::jsonb,
 'active')
ON CONFLICT (id) DO NOTHING;

-- Casting 3: Fashion show (Employer 1)
INSERT INTO castings (id, creator_id, title, description, city,
                      pay_min, pay_max, pay_type, date_from,
                      requirements, status)
VALUES
('cccc3333-3333-3333-3333-333333333333',
 '33333333-3333-3333-3333-333333333333',
 'Models for collection show',
 'Luxury Brand is looking for models for the spring-summer 2025 collection show.',
 'Astana',
 30000, 50000, 'negotiable',
 NOW() + interval '15 days',
 '{"gender": "female", "age_min": 18, "age_max": 30, "height_min": 175}'::jsonb,
 'active')
ON CONFLICT (id) DO NOTHING;

-- Casting 4: TFP free (Employer 2)
INSERT INTO castings (id, creator_id, title, description, city,
                      pay_type, date_from, status)
VALUES
('cccc4444-4444-4444-4444-444444444444',
 '44444444-4444-4444-4444-444444444444',
 'TFP Portfolio photoshoot',
 'Free photoshoot for portfolio. Perfect for beginner models! Get quality photos.',
 'Almaty',
 'free',
 NOW() + interval '3 days',
 'active')
ON CONFLICT (id) DO NOTHING;

-- Casting 5: Cosmetics ad (Employer 1)
INSERT INTO castings (id, creator_id, title, description, city,
                      pay_min, pay_max, pay_type, date_from,
                      requirements, status)
VALUES
('cccc5555-5555-5555-5555-555555555555',
 '33333333-3333-3333-3333-333333333333',
 'Cosmetics advertising shoot',
 'Beauty Co is looking for a model for new cosmetics line advertising. Close-up face shots, clean skin important.',
 'Almaty',
 20000, 25000, 'fixed',
 NOW() + interval '7 days',
 '{"gender": "female", "age_min": 18, "age_max": 35}'::jsonb,
 'active')
ON CONFLICT (id) DO NOTHING;

-- Casting 6: Fashion brand catalog (Employer 2)
INSERT INTO castings (id, creator_id, title, description, city,
                      pay_min, pay_max, pay_type, date_from,
                      requirements, status)
VALUES
('cccc6666-6666-6666-6666-666666666666',
 '44444444-4444-4444-4444-444444444444',
 'Fashion brand photoshoot',
 'Fashion Studio is looking for models for new clothing collection catalog shoot. Studio shoot, full day.',
 'Almaty',
 15000, 25000, 'negotiable',
 NOW() + interval '2 days',
 '{"age_min": 18, "age_max": 30}'::jsonb,
 'active')
ON CONFLICT (id) DO NOTHING;

-- ============================================
-- 4. SUBSCRIPTIONS (give users free plan)
-- ============================================

INSERT INTO subscriptions (id, user_id, plan_id, started_at, status, billing_period)
VALUES 
    (gen_random_uuid(), '11111111-1111-1111-1111-111111111111', 'free', NOW(), 'active', 'monthly'),
    (gen_random_uuid(), '22222222-2222-2222-2222-222222222222', 'free', NOW(), 'active', 'monthly'),
    (gen_random_uuid(), '33333333-3333-3333-3333-333333333333', 'free', NOW(), 'active', 'monthly'),
    (gen_random_uuid(), '44444444-4444-4444-4444-444444444444', 'free', NOW(), 'active', 'monthly')
ON CONFLICT DO NOTHING;

-- ============================================
-- Done!
-- ============================================
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Seed data created successfully!';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Test accounts (password: password123):';
    RAISE NOTICE '  - test1@test.com (model)';
    RAISE NOTICE '  - model2@test.com (model)';
    RAISE NOTICE '  - employer1@test.com (employer)';
    RAISE NOTICE '  - employer2@test.com (employer)';
    RAISE NOTICE '========================================';
END $$;
