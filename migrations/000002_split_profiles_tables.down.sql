DO $$
    BEGIN
        IF EXISTS (
            SELECT 1
            FROM information_schema.tables
            WHERE table_schema = 'public'
              AND table_name = 'responses'
        ) THEN
            ALTER TABLE responses
                DROP CONSTRAINT IF EXISTS responses_profile_id_fkey;

            ALTER TABLE responses
                ADD CONSTRAINT responses_profile_id_fkey
                    FOREIGN KEY (profile_id)
                        REFERENCES profiles(id)
                        ON DELETE CASCADE;
        END IF;
    END $$;

DROP TABLE IF EXISTS admin_profiles;
DROP TABLE IF EXISTS employer_profiles;
DROP TABLE IF EXISTS model_profiles;
