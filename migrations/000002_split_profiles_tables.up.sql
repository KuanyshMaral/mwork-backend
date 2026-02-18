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

-- If model_profiles existed from old migration (id,name,email only), evolve schema in-place
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS user_id UUID;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS bio TEXT;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS age INTEGER;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS height DOUBLE PRECISION;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS weight DOUBLE PRECISION;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS gender VARCHAR(20);
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS clothing_size VARCHAR(20);
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS shoe_size VARCHAR(20);
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS experience INTEGER;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS hourly_rate DOUBLE PRECISION;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS city VARCHAR(100);
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS country VARCHAR(100);
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS languages JSONB;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS categories JSONB;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS skills JSONB;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS barter_accepted BOOLEAN;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS accept_remote_work BOOLEAN;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS travel_cities JSONB;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS visibility VARCHAR(20);
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS profile_views INTEGER;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS rating DOUBLE PRECISION;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS total_reviews INTEGER;
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS is_public BOOLEAN;
ALTER TABLE model_profiles ALTER COLUMN languages SET DEFAULT '[]'::jsonb;
ALTER TABLE model_profiles ALTER COLUMN categories SET DEFAULT '[]'::jsonb;
ALTER TABLE model_profiles ALTER COLUMN skills SET DEFAULT '[]'::jsonb;
ALTER TABLE model_profiles ALTER COLUMN barter_accepted SET DEFAULT FALSE;
ALTER TABLE model_profiles ALTER COLUMN accept_remote_work SET DEFAULT FALSE;
ALTER TABLE model_profiles ALTER COLUMN travel_cities SET DEFAULT '[]'::jsonb;
ALTER TABLE model_profiles ALTER COLUMN visibility SET DEFAULT 'public';
ALTER TABLE model_profiles ALTER COLUMN profile_views SET DEFAULT 0;
ALTER TABLE model_profiles ALTER COLUMN rating SET DEFAULT 0;
ALTER TABLE model_profiles ALTER COLUMN total_reviews SET DEFAULT 0;
ALTER TABLE model_profiles ALTER COLUMN is_public SET DEFAULT TRUE;
UPDATE model_profiles SET languages = '[]'::jsonb WHERE languages IS NULL;
UPDATE model_profiles SET categories = '[]'::jsonb WHERE categories IS NULL;
UPDATE model_profiles SET skills = '[]'::jsonb WHERE skills IS NULL;
UPDATE model_profiles SET barter_accepted = FALSE WHERE barter_accepted IS NULL;
UPDATE model_profiles SET accept_remote_work = FALSE WHERE accept_remote_work IS NULL;
UPDATE model_profiles SET travel_cities = '[]'::jsonb WHERE travel_cities IS NULL;
UPDATE model_profiles SET visibility = 'public' WHERE visibility IS NULL;
UPDATE model_profiles SET profile_views = 0 WHERE profile_views IS NULL;
UPDATE model_profiles SET rating = 0 WHERE rating IS NULL;
UPDATE model_profiles SET total_reviews = 0 WHERE total_reviews IS NULL;
UPDATE model_profiles SET is_public = TRUE WHERE is_public IS NULL;

-- Best-effort user_id backfill for legacy rows created with email
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'model_profiles'
          AND column_name = 'email'
    ) THEN
        EXECUTE $sql$
            UPDATE model_profiles mp
            SET user_id = u.id
            FROM users u
            WHERE mp.user_id IS NULL
              AND mp.email IS NOT NULL
              AND u.email = mp.email
        $sql$;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_model_profiles_user_id_unique ON model_profiles(user_id);

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
-- After split, responses should reference model_profiles instead of legacy profiles.
DO $$
    BEGIN
        IF EXISTS (
            SELECT 1
            FROM information_schema.tables
            WHERE table_schema = 'public'
              AND table_name = 'responses'
        ) THEN
            ALTER TABLE responses
                DROP CONSTRAINT IF EXISTS responses_profile_id_fkey;

            ALTER TABLE responses
                ADD CONSTRAINT responses_profile_id_fkey
                    FOREIGN KEY (profile_id)
                        REFERENCES model_profiles(id)
                        ON DELETE CASCADE;
        END IF;
    END $$;