-- migrations/000002_create_profiles.up.sql
-- Profiles table for models and employers

CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('model', 'employer')),
    
    -- Common fields
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    bio TEXT,
    avatar_url VARCHAR(500),
    city VARCHAR(100),
    phone VARCHAR(20),
    
    -- Model-specific fields
    height_cm INTEGER CHECK (height_cm IS NULL OR (height_cm >= 100 AND height_cm <= 250)),
    weight_kg INTEGER CHECK (weight_kg IS NULL OR (weight_kg >= 30 AND weight_kg <= 200)),
    age INTEGER CHECK (age IS NULL OR (age >= 14 AND age <= 100)),
    gender VARCHAR(20) CHECK (gender IS NULL OR gender IN ('male', 'female', 'other')),
    clothing_size VARCHAR(10),
    shoe_size VARCHAR(10),
    hourly_rate DECIMAL(10,2) CHECK (hourly_rate IS NULL OR hourly_rate >= 0),
    experience_years INTEGER DEFAULT 0 CHECK (experience_years >= 0),
    languages JSONB DEFAULT '[]',
    
    -- Employer-specific fields
    company_name VARCHAR(200),
    website VARCHAR(500),
    instagram VARCHAR(100),
    
    -- Settings & stats
    is_public BOOLEAN DEFAULT TRUE,
    view_count INTEGER DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_profiles_user_id ON profiles(user_id);
CREATE INDEX idx_profiles_type ON profiles(type);
CREATE INDEX idx_profiles_city ON profiles(city) WHERE is_public = true;
CREATE INDEX idx_profiles_city_type ON profiles(city, type) WHERE is_public = true;

-- Model-specific search index
CREATE INDEX idx_profiles_model_params ON profiles(city, height_cm, age, gender) 
    WHERE type = 'model' AND is_public = true;

-- Trigger for updated_at
CREATE TRIGGER profiles_updated_at
    BEFORE UPDATE ON profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Comments
COMMENT ON TABLE profiles IS 'User profiles for models and employers';
COMMENT ON COLUMN profiles.type IS 'Profile type: model or employer';
COMMENT ON COLUMN profiles.languages IS 'Array of language codes, e.g. ["russian", "kazakh", "english"]';
