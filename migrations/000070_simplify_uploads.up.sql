-- 000069_simplify_uploads.up.sql
-- Радикальное упрощение: убираем 2-phase upload, staging, presign.
-- Переходим на "Глупый склад": 7 колонок, локальное хранение.

-- Удаляем старую таблицу со всеми зависимостями (photos.upload_id, messages.attachment_upload_id)
DROP TABLE IF EXISTS photos CASCADE;
DROP TABLE IF EXISTS uploads CASCADE;

-- Создаём новую плоскую таблицу
CREATE TABLE uploads (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_path     VARCHAR(500) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    mime_type     VARCHAR(100) NOT NULL,
    size_bytes    BIGINT NOT NULL CHECK (size_bytes > 0),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индекс для быстрого поиска файлов по автору
CREATE INDEX idx_uploads_author_id ON uploads(author_id);
CREATE INDEX idx_uploads_created_at ON uploads(created_at DESC);

COMMENT ON TABLE uploads IS 'Flat file storage registry. Upload module knows nothing about business logic.';
COMMENT ON COLUMN uploads.author_id IS 'User who uploaded the file. ON DELETE CASCADE removes all user files from registry.';
COMMENT ON COLUMN uploads.file_path IS 'Logical path relative to storage root: {author_id}/{uuid}.ext';
