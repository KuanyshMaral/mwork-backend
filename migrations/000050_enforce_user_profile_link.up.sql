-- 000050_enforce_user_profile_link.up.sql
-- This migration is a no-op: model_profiles and employer_profiles already enforce
-- a strict one-to-one relation with users via UNIQUE REFERENCES users(id) ON DELETE CASCADE
-- defined in 000002_split_profiles_tables.up.sql.
-- The legacy profiles table no longer exists.
SELECT 1;
