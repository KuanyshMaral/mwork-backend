-- Добавляем колонку plan_id в таблицу payments
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS plan_id UUID;

-- Добавляем колонку updated_at в таблицу payments
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP;

-- Добавляем колонку promotion_id в таблицу payments
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS promotion_id UUID;

-- Добавляем индекс для promotion_id (если необходимо)
CREATE INDEX IF NOT EXISTS idx_payments_promotion_id
    ON payments(promotion_id);

-- Добавляем комментарии для колонок
COMMENT ON COLUMN payments.plan_id IS 'ID плана для платежа';
COMMENT ON COLUMN payments.updated_at IS 'Дата и время последнего обновления записи';
COMMENT ON COLUMN payments.promotion_id IS 'ID акции, связанной с платежом';
