-- 000032_add_travel_availability.up.sql

-- Добавляем колонку для доступных городов
ALTER TABLE profiles ADD COLUMN IF NOT EXISTS travel_cities TEXT[] DEFAULT '{}';

-- Добавляем комментарий к колонке
COMMENT ON COLUMN profiles.travel_cities IS 'Cities model is willing to travel to for work';
