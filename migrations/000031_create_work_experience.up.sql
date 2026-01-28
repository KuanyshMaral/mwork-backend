-- 000031_create_work_experience.up.sql

-- Создаем таблицу для опыта работы моделей
CREATE TABLE IF NOT EXISTS work_experiences (
                                                id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                                profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
                                                title VARCHAR(200) NOT NULL,
                                                company VARCHAR(200),
                                                role VARCHAR(100),
                                                year INTEGER,
                                                description TEXT,
                                                created_at TIMESTAMP DEFAULT NOW()
);

-- Создаем индекс для быстрого поиска по profile_id
CREATE INDEX IF NOT EXISTS idx_work_experiences_profile_id ON work_experiences(profile_id);

-- Добавляем комментарий к таблице
COMMENT ON TABLE work_experiences IS 'Model work history and portfolio projects';
