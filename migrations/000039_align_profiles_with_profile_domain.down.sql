-- 000037_align_profiles_with_profile_domain.down.sql

-- Reverse changes (safe drops)

ALTER TABLE profiles DROP COLUMN IF EXISTS verified_at;
ALTER TABLE profiles DROP COLUMN IF EXISTS is_verified;
ALTER TABLE profiles DROP COLUMN IF EXISTS contact_phone;
ALTER TABLE profiles DROP COLUMN IF EXISTS contact_person;
ALTER TABLE profiles DROP COLUMN IF EXISTS company_type;

ALTER TABLE profiles DROP COLUMN IF EXISTS total_reviews;
ALTER TABLE profiles DROP COLUMN IF EXISTS accept_remote_work;
ALTER TABLE profiles DROP COLUMN IF EXISTS barter_accepted;

ALTER TABLE profiles DROP COLUMN IF EXISTS skills;
ALTER TABLE profiles DROP COLUMN IF EXISTS categories;

ALTER TABLE profiles DROP COLUMN IF EXISTS country;
ALTER TABLE profiles DROP COLUMN IF EXISTS description;

ALTER TABLE profiles DROP COLUMN IF EXISTS travel_cities;
