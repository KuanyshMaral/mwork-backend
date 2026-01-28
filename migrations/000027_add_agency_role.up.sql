-- 000027_add_agency_role.up.sql

-- Удаляем старое ограничение для role, если оно существует
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;

-- Добавляем новое ограничение для поддержки роли 'agency'
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('model', 'employer', 'agency', 'admin'));

-- Добавляем комментарий к ограничению
COMMENT ON CONSTRAINT users_role_check ON users IS 'Valid user roles including modeling agencies';
