-- Ensure strict one-to-one relation between users and profiles.

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'profiles_user_id_fkey'
    ) THEN
        ALTER TABLE profiles
            ADD CONSTRAINT profiles_user_id_fkey
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'profiles_user_id_key'
    ) THEN
        ALTER TABLE profiles
            ADD CONSTRAINT profiles_user_id_key UNIQUE (user_id);
    END IF;
END $$;
