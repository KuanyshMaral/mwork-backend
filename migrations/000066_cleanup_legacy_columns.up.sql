-- Remove legacy JSONB column that is no longer used
ALTER TABLE castings DROP COLUMN IF EXISTS requirements;
