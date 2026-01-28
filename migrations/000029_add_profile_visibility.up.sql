-- 000029_add_profile_visibility.up.sql

-- Добавляем колонку для видимости профиля
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS visibility VARCHAR(20) DEFAULT 'public';

-- Добавляем проверку на допустимые значения для видимости
ALTER TABLE profiles ADD CONSTRAINT profiles_visibility_check
    CHECK (visibility IN ('public', 'link_only', 'hidden'));

-- Добавляем комментарий к колонке
COMMENT ON COLUMN profiles.visibility IS 'Profile visibility: public, link_only, or hidden';
