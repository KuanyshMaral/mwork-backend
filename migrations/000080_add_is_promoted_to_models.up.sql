-- Migration: Add is_promoted to model_profiles
-- Purpose: Fast lookup and sorting for promoted models without joining profile_promotions

ALTER TABLE model_profiles ADD COLUMN IF NOT EXISTS is_promoted BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_model_profiles_is_promoted ON model_profiles(is_promoted);
