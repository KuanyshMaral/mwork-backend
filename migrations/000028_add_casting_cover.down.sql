-- 000028_add_casting_cover.down.sql

-- Удаляем колонку cover_image_url
ALTER TABLE castings DROP COLUMN IF EXISTS cover_image_url;
