UPDATE uploads
SET size = 1
WHERE size IS NULL;

ALTER TABLE uploads DROP CONSTRAINT IF EXISTS uploads_size_check;

ALTER TABLE uploads
    ADD CONSTRAINT uploads_size_check CHECK (size > 0);

ALTER TABLE uploads
    ALTER COLUMN size SET NOT NULL;
