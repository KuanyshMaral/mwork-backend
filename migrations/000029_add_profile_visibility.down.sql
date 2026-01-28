-- 000029_add_profile_visibility.down.sql

-- Удаляем колонку visibility
ALTER TABLE profiles DROP COLUMN IF EXISTS visibility;

-- Удаляем проверку для видимости
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_visibility_check;
