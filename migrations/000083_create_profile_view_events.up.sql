CREATE TABLE IF NOT EXISTS profile_view_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES model_profiles(id) ON DELETE CASCADE,
    viewer_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_profile_view_events_profile_viewed_at
    ON profile_view_events(profile_id, viewed_at DESC);

CREATE INDEX IF NOT EXISTS idx_profile_view_events_profile_viewer
    ON profile_view_events(profile_id, viewer_user_id, viewed_at DESC)
    WHERE viewer_user_id IS NOT NULL;
