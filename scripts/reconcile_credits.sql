-- Read-only reconciliation query for credit consistency checks.
SELECT
    u.id AS user_id,
    u.credit_balance AS stored_balance,
    COALESCE(SUM(ct.amount_delta), 0)::INTEGER AS computed_balance,
    (u.credit_balance - COALESCE(SUM(ct.amount_delta), 0)::INTEGER) AS diff
FROM users u
LEFT JOIN credit_transactions ct ON ct.user_id = u.id
GROUP BY u.id, u.credit_balance
HAVING u.credit_balance <> COALESCE(SUM(ct.amount_delta), 0)::INTEGER
ORDER BY ABS(u.credit_balance - COALESCE(SUM(ct.amount_delta), 0)::INTEGER) DESC, u.id;

-- Optional hardening (run with your DB role names):
-- REVOKE UPDATE, DELETE ON TABLE credit_transactions FROM <app_role>;
