-- 000031_create_work_experience.down.sql

-- Удаляем таблицу work_experiences и индекс
DROP INDEX IF EXISTS idx_work_experiences_profile_id;
DROP TABLE IF EXISTS work_experiences;
