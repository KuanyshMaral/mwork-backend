CREATE TABLE model_profiles (
                                id UUID PRIMARY KEY,
                                name VARCHAR(100),
                                email VARCHAR(100),
                                created_at TIMESTAMP DEFAULT NOW(),
                                updated_at TIMESTAMP DEFAULT NOW()
);

-- Добавляем комментарии для колонок
COMMENT ON COLUMN model_profiles.name IS 'Имя модели';
COMMENT ON COLUMN model_profiles.email IS 'Email модели';
COMMENT ON COLUMN model_profiles.created_at IS 'Дата и время создания профиля модели';
COMMENT ON COLUMN model_profiles.updated_at IS 'Дата и время последнего обновления профиля модели';
