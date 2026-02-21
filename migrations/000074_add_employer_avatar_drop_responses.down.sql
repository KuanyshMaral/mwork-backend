-- Recreate legacy responses table
CREATE TABLE IF NOT EXISTS responses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    casting_id UUID NOT NULL REFERENCES castings(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES model_profiles(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    cover_letter TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_responses_casting_id ON responses(casting_id);
CREATE INDEX IF NOT EXISTS idx_responses_profile_id ON responses(profile_id);
CREATE INDEX IF NOT EXISTS idx_responses_user_id ON responses(user_id);
CREATE INDEX IF NOT EXISTS idx_responses_status ON responses(status);

-- Remove employer avatar
DROP INDEX IF EXISTS idx_employer_profiles_avatar_upload_id;
ALTER TABLE employer_profiles DROP COLUMN IF EXISTS avatar_upload_id;
