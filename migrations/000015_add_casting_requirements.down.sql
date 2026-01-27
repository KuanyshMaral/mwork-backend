-- Rollback: Remove casting requirements and saved castings

DROP TABLE IF EXISTS saved_castings;

DROP INDEX IF EXISTS idx_castings_event;
DROP INDEX IF EXISTS idx_castings_deadline;
DROP INDEX IF EXISTS idx_castings_urgent;
DROP INDEX IF EXISTS idx_castings_height;
DROP INDEX IF EXISTS idx_castings_age;
DROP INDEX IF EXISTS idx_castings_gender;

ALTER TABLE castings DROP COLUMN IF EXISTS views_count;
ALTER TABLE castings DROP COLUMN IF EXISTS is_urgent;
ALTER TABLE castings DROP COLUMN IF EXISTS deadline_at;
ALTER TABLE castings DROP COLUMN IF EXISTS event_location;
ALTER TABLE castings DROP COLUMN IF EXISTS event_datetime;
ALTER TABLE castings DROP COLUMN IF EXISTS work_type;
ALTER TABLE castings DROP COLUMN IF EXISTS shoe_sizes;
ALTER TABLE castings DROP COLUMN IF EXISTS clothing_sizes;
ALTER TABLE castings DROP COLUMN IF EXISTS required_languages;
ALTER TABLE castings DROP COLUMN IF EXISTS required_experience;
ALTER TABLE castings DROP COLUMN IF EXISTS max_weight;
ALTER TABLE castings DROP COLUMN IF EXISTS min_weight;
ALTER TABLE castings DROP COLUMN IF EXISTS max_height;
ALTER TABLE castings DROP COLUMN IF EXISTS min_height;
ALTER TABLE castings DROP COLUMN IF EXISTS max_age;
ALTER TABLE castings DROP COLUMN IF EXISTS min_age;
ALTER TABLE castings DROP COLUMN IF EXISTS required_gender;
