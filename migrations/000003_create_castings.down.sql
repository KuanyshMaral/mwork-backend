-- migrations/000003_create_castings.down.sql

DROP TRIGGER IF EXISTS castings_updated_at ON castings;
DROP TABLE IF EXISTS castings;
