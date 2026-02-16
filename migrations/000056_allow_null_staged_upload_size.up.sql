ALTER TABLE uploads
    ALTER COLUMN size DROP NOT NULL;

ALTER TABLE uploads DROP CONSTRAINT IF EXISTS uploads_size_check;

ALTER TABLE uploads
    ADD CONSTRAINT uploads_size_check CHECK (size IS NULL OR size > 0);
