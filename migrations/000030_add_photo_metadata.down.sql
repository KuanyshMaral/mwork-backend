-- 000030_add_photo_metadata.down.sql

-- Удаляем колонки для метаданных фотографий
ALTER TABLE photos DROP COLUMN IF EXISTS caption;
ALTER TABLE photos DROP COLUMN IF EXISTS project_name;
ALTER TABLE photos DROP COLUMN IF EXISTS brand;
ALTER TABLE photos DROP COLUMN IF EXISTS year;
