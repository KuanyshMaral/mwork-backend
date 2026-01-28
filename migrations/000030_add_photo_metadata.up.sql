-- 000030_add_photo_metadata.up.sql

-- Добавляем колонки для метаданных фотографий
ALTER TABLE photos ADD COLUMN IF NOT EXISTS caption TEXT;
ALTER TABLE photos ADD COLUMN IF NOT EXISTS project_name VARCHAR(200);
ALTER TABLE photos ADD COLUMN IF NOT EXISTS brand VARCHAR(200);
ALTER TABLE photos ADD COLUMN IF NOT EXISTS year INTEGER;

-- Добавляем комментарии к колонкам
COMMENT ON COLUMN photos.caption IS 'Photo caption or description';
COMMENT ON COLUMN photos.project_name IS 'Project or campaign name';
COMMENT ON COLUMN photos.brand IS 'Brand or client name';
COMMENT ON COLUMN photos.year IS 'Project year';
