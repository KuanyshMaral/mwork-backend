-- Migration: Add tags to castings
-- Purpose: Support dynamic user-defined categories/tags for castings (e.g. "Photoshoot", "Fashion Show")

ALTER TABLE castings ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';

-- GIN index for fast filtering by tags (e.g. WHERE tags @> '{Photoshoot}')
CREATE INDEX IF NOT EXISTS idx_castings_tags ON castings USING GIN (tags);

COMMENT ON COLUMN castings.tags IS 'User-defined array of tags/categories (e.g. ["Photoshoot", "Fashion Show"])';
