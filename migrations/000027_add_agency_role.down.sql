-- 000027_add_agency_role.down.sql

-- Удаляем новое ограничение для роли 'agency'
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;

-- Добавляем старое ограничение для ролей, без 'agency'
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('model', 'employer', 'admin'));

-- Комментарий можно оставить, если нужно.
COMMENT ON CONSTRAINT users_role_check ON users IS 'Valid user roles excluding modeling agencies';
