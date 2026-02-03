-- 000043_fix_admin_seed_password.up.sql
-- Fix incorrect default admin password hash for admin@mwork.kz

UPDATE admin_users
SET password_hash = '$2a$12$ob/WOA675I7wHxM/a9aQKexapuOA2ll28eCwHZhG8dxdEUEWL8kH.'
WHERE email = 'admin@mwork.kz'
  AND password_hash = '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4tJxzPXzLqZKJKWe';
