-- Создание таблицы casting_responses
CREATE TABLE casting_responses (
                                   id UUID PRIMARY KEY,
                                   casting_id UUID NOT NULL,
                                   model_id UUID NOT NULL,
                                   message TEXT,
                                   proposed_rate DECIMAL(10,2),
                                   status VARCHAR(50),
                                   created_at TIMESTAMP DEFAULT NOW(),
                                   updated_at TIMESTAMP DEFAULT NOW(),
                                   user_id UUID, -- добавляем user_id для связи с пользователем
                                   accepted_at TIMESTAMP, -- дата принятия отклика
                                   rejected_at TIMESTAMP -- дата отклонения отклика
);

-- Создание индексов для улучшения производительности
CREATE INDEX IF NOT EXISTS idx_casting_responses_casting_id ON casting_responses(casting_id);
CREATE INDEX IF NOT EXISTS idx_casting_responses_model_id ON casting_responses(model_id);
CREATE INDEX IF NOT EXISTS idx_casting_responses_user_id ON casting_responses(user_id);

-- Добавляем комментарии для колонок
COMMENT ON COLUMN casting_responses.casting_id IS 'ID кастинга';
COMMENT ON COLUMN casting_responses.model_id IS 'ID модели';
COMMENT ON COLUMN casting_responses.status IS 'Статус отклика';
COMMENT ON COLUMN casting_responses.created_at IS 'Дата и время создания';
COMMENT ON COLUMN casting_responses.updated_at IS 'Дата и время последнего обновления';
COMMENT ON COLUMN casting_responses.user_id IS 'ID пользователя, оставившего отклик';
COMMENT ON COLUMN casting_responses.accepted_at IS 'Дата и время принятия отклика';
COMMENT ON COLUMN casting_responses.rejected_at IS 'Дата и время отклонения отклика';
