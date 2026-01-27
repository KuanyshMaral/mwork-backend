-- Rollback: Remove profile measurements and social links

DROP TABLE IF EXISTS profile_social_links;

DROP INDEX IF EXISTS idx_profiles_eye;
DROP INDEX IF EXISTS idx_profiles_hair;
DROP INDEX IF EXISTS idx_profiles_measurements;
DROP INDEX IF EXISTS idx_profiles_specializations;

ALTER TABLE profiles DROP COLUMN IF EXISTS profile_views;
ALTER TABLE profiles DROP COLUMN IF EXISTS completed_jobs;
ALTER TABLE profiles DROP COLUMN IF EXISTS total_earnings;
ALTER TABLE profiles DROP COLUMN IF EXISTS specializations;
ALTER TABLE profiles DROP COLUMN IF EXISTS skin_tone;
ALTER TABLE profiles DROP COLUMN IF EXISTS eye_color;
ALTER TABLE profiles DROP COLUMN IF EXISTS hair_color;
ALTER TABLE profiles DROP COLUMN IF EXISTS shoe_size;
ALTER TABLE profiles DROP COLUMN IF EXISTS hips_cm;
ALTER TABLE profiles DROP COLUMN IF EXISTS waist_cm;
ALTER TABLE profiles DROP COLUMN IF EXISTS bust_cm;
