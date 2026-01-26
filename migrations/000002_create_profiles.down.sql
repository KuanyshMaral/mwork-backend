-- migrations/000002_create_profiles.down.sql

DROP TRIGGER IF EXISTS profiles_updated_at ON profiles;
DROP TABLE IF EXISTS profiles;
