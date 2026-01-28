-- 000032_add_travel_availability.down.sql

-- Удаляем колонку travel_cities
ALTER TABLE profiles DROP COLUMN IF EXISTS travel_cities;
