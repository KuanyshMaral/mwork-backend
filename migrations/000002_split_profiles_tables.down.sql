-- 000002_split_profiles_tables.down.sql
-- Rollback: drop the split profile tables.
-- Note: the legacy profiles table is NOT recreated here since it was dropped
-- as part of the data migration. A full rollback would require restoring from backup.

DROP TABLE IF EXISTS admin_profiles;
DROP TABLE IF EXISTS employer_profiles;
DROP TABLE IF EXISTS model_profiles;
