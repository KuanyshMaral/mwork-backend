-- 000028_add_casting_cover.up.sql

-- Добавляем колонку для URL изображения обложки кастинга
ALTER TABLE castings ADD COLUMN IF NOT EXISTS cover_image_url VARCHAR(500);

-- Добавляем комментарий к колонке
COMMENT ON COLUMN castings.cover_image_url IS 'Cover image URL from Cloudflare R2 storage';
