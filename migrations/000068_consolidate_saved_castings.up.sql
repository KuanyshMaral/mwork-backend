-- Migration 000067: Consolidate saved_castings into favorites
-- Migrate all existing data, then drop the legacy table.

-- Step 1: Move existing saved_castings rows into the polymorphic favorites table
INSERT INTO favorites (id, user_id, entity_type, entity_id, created_at)
SELECT
    gen_random_uuid(),
    sc.user_id,
    'casting',
    sc.casting_id,
    sc.created_at
FROM saved_castings sc
WHERE NOT EXISTS (
    -- Avoid duplicates if someone already has it in favorites
    SELECT 1 FROM favorites f
    WHERE f.user_id = sc.user_id
      AND f.entity_type = 'casting'
      AND f.entity_id = sc.casting_id
);

-- Step 2: Drop the legacy table
DROP TABLE IF EXISTS saved_castings;
