-- Удаляем колонку plan_id из таблицы payments
ALTER TABLE payments
    DROP COLUMN IF EXISTS plan_id;

-- Удаляем колонку updated_at из таблицы payments
ALTER TABLE payments
    DROP COLUMN IF EXISTS updated_at;

-- Удаляем колонку promotion_id из таблицы payments
ALTER TABLE payments
    DROP COLUMN IF EXISTS promotion_id;

-- Удаляем индекс для promotion_id
DROP INDEX IF EXISTS idx_payments_promotion_id;
