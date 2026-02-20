-- 000032_add_travel_availability.up.sql

-- Add travel_cities column to model_profiles (models can specify cities they travel to)
ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS travel_cities TEXT[] DEFAULT '{}';

-- Add comment to column
COMMENT ON COLUMN model_profiles.travel_cities IS 'Cities model is willing to travel to for work';
