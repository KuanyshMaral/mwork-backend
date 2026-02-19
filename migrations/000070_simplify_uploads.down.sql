-- 000069_simplify_uploads.down.sql
-- Откат невозможен без потери данных, но структура восстанавливается.
DROP TABLE IF EXISTS uploads CASCADE;
