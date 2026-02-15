-- Split legacy profiles table into role-specific profile tables.

CREATE TABLE IF NOT EXISTS model_profiles (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(200),
    bio TEXT,
    description TEXT,
    age INTEGER,
    height DOUBLE PRECISION,
    weight DOUBLE PRECISION,
    gender VARCHAR(20),
    clothing_size VARCHAR(20),
    shoe_size VARCHAR(20),
    experience INTEGER,
    hourly_rate DOUBLE PRECISION,
    city VARCHAR(100),
    country VARCHAR(100),
    languages JSONB NOT NULL DEFAULT '[]'::jsonb,
    categories JSONB NOT NULL DEFAULT '[]'::jsonb,
    skills JSONB NOT NULL DEFAULT '[]'::jsonb,
    barter_accepted BOOLEAN NOT NULL DEFAULT FALSE,
    accept_remote_work BOOLEAN NOT NULL DEFAULT FALSE,
    travel_cities JSONB NOT NULL DEFAULT '[]'::jsonb,
    visibility VARCHAR(20) DEFAULT 'public',
    profile_views INTEGER NOT NULL DEFAULT 0,
    rating DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_reviews INTEGER NOT NULL DEFAULT 0,
    is_public BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS employer_profiles (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    company_name VARCHAR(255) NOT NULL DEFAULT '',
    company_type VARCHAR(100),
    description TEXT,
    website VARCHAR(500),
    contact_person VARCHAR(200),
    contact_phone VARCHAR(20),
    city VARCHAR(100),
    country VARCHAR(100),
    rating DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_reviews INTEGER NOT NULL DEFAULT 0,
    castings_posted INTEGER NOT NULL DEFAULT 0,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS admin_profiles (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL UNIQUE,
    name VARCHAR(255),
    role VARCHAR(50),
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO model_profiles (
    id, user_id, name, bio, description, age, height, weight, gender,
    clothing_size, shoe_size, experience, hourly_rate, city, country,
    languages, categories, skills, barter_accepted, accept_remote_work,
    travel_cities, visibility, profile_views, rating, total_reviews, is_public,
    created_at, updated_at
)
SELECT
    id, user_id, first_name, bio, description, age, height_cm, weight_kg, gender,
    clothing_size, shoe_size, experience_years, hourly_rate, city, country,
    COALESCE(languages, '[]'::jsonb), COALESCE(categories, '[]'::jsonb), COALESCE(skills, '[]'::jsonb),
    COALESCE(barter_accepted, FALSE), COALESCE(accept_remote_work, FALSE),
    COALESCE(travel_cities, '[]'::jsonb), COALESCE(visibility, 'public'),
    COALESCE(view_count, 0), COALESCE(rating, 0), COALESCE(total_reviews, 0), COALESCE(is_public, TRUE),
    created_at, updated_at
FROM profiles
WHERE type = 'model'
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO employer_profiles (
    id, user_id, company_name, company_type, description, website,
    contact_person, contact_phone, city, country,
    rating, total_reviews, is_verified, verified_at,
    created_at, updated_at
)
SELECT
    id, user_id, COALESCE(company_name, ''), company_type, description, website,
    contact_person, contact_phone, city, country,
    COALESCE(rating, 0), COALESCE(total_reviews, 0), COALESCE(is_verified, FALSE), verified_at,
    created_at, updated_at
FROM profiles
WHERE type = 'employer'
ON CONFLICT (user_id) DO NOTHING;
