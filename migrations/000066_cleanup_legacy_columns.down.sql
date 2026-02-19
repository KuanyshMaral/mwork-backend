-- Restore legacy column
ALTER TABLE castings ADD COLUMN IF NOT EXISTS requirements JSONB DEFAULT '{}';

COMMENT ON COLUMN castings.requirements IS 'Legacy Flexible JSON structure for model requirements';
