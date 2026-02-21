package credit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const queryTimeout = 3 * time.Second

type Repository interface {
	Deduct(ctx context.Context, userID string, amount int, meta TxMeta) error
	DeductTx(ctx context.Context, tx *sqlx.Tx, userID string, amount int, meta TxMeta) error
	Add(ctx context.Context, userID string, amount int, txType string, meta TxMeta) error
	GetBalance(ctx context.Context, userID string) (int, error)
	ListTransactions(ctx context.Context, userID string, pagination Pagination) ([]CreditTransaction, error)
	SearchTransactions(ctx context.Context, filters SearchFilters) ([]CreditTransaction, error)
}

// CreditRepository provides credit ledger and balance operations.
type CreditRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *CreditRepository {
	return &CreditRepository{db: db}
}

func (r *CreditRepository) Deduct(ctx context.Context, userID string, amount int, meta TxMeta) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	ctx2, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTxx(ctx2, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("%w: begin tx", ErrInternal)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx2, `
		UPDATE users
		SET credit_balance = credit_balance - $2
		WHERE id = $1 AND credit_balance >= $2
	`, userID, amount)
	if err != nil {
		return fmt.Errorf("%w: update user balance", ErrInternal)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: rows affected", ErrInternal)
	}
	if rows == 0 {
		return ErrInsufficientCredits
	}

	if err := r.insertLedger(ctx2, tx, userID, -amount, string(TxTypeDeduction), meta); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit tx", ErrInternal)
	}

	return nil
}

// DeductTx deducts credits within an external transaction using FOR UPDATE row lock.
// This method does NOT commit or rollback the transaction — the caller is responsible.
func (r *CreditRepository) DeductTx(ctx context.Context, tx *sqlx.Tx, userID string, amount int, meta TxMeta) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	// FOR UPDATE lock on the user's credit balance row
	var balance int
	err := tx.QueryRowContext(ctx, `SELECT credit_balance FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return fmt.Errorf("%w: lock user row", ErrInternal)
	}

	if balance < amount {
		return ErrInsufficientCredits
	}

	// Deduct balance
	_, err = tx.ExecContext(ctx, `UPDATE users SET credit_balance = credit_balance - $2 WHERE id = $1`, userID, amount)
	if err != nil {
		return fmt.Errorf("%w: update user balance", ErrInternal)
	}

	// Write ledger entry within the same transaction
	if err := r.insertLedger(ctx, tx, userID, -amount, string(TxTypeDeduction), meta); err != nil {
		return err
	}

	return nil
}

func (r *CreditRepository) Add(ctx context.Context, userID string, amount int, txType string, meta TxMeta) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	ctx2, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTxx(ctx2, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("%w: begin tx", ErrInternal)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx2, `
		UPDATE users
		SET credit_balance = credit_balance + $2
		WHERE id = $1
	`, userID, amount)
	if err != nil {
		return fmt.Errorf("%w: update user balance", ErrInternal)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: rows affected", ErrInternal)
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	if err := r.insertLedger(ctx2, tx, userID, amount, txType, meta); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit tx", ErrInternal)
	}

	return nil
}

func (r *CreditRepository) GetBalance(ctx context.Context, userID string) (int, error) {
	ctx2, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var balance int
	err := r.db.GetContext(ctx2, &balance, `SELECT credit_balance FROM users WHERE id = $1`, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("%w: get balance", ErrInternal)
	}

	return balance, nil
}

func (r *CreditRepository) ListTransactions(ctx context.Context, userID string, pagination Pagination) ([]CreditTransaction, error) {
	ctx2, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit := pagination.Limit
	if limit <= 0 {
		limit = 20
	}

	transactions := make([]CreditTransaction, 0)
	err := r.db.SelectContext(ctx2, &transactions, `
		SELECT id, user_id, amount_delta, tx_type, related_entity_type, related_entity_id, description, created_at
		FROM credit_transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, pagination.Offset)
	if err != nil {
		return nil, fmt.Errorf("%w: list transactions", ErrInternal)
	}

	return transactions, nil
}

func (r *CreditRepository) SearchTransactions(ctx context.Context, filters SearchFilters) ([]CreditTransaction, error) {
	ctx2, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	base := `
		SELECT id, user_id, amount_delta, tx_type, related_entity_type, related_entity_id, description, created_at
		FROM credit_transactions
		WHERE 1=1`
	args := make([]interface{}, 0, 8)
	idx := 1

	if filters.UserID != nil && *filters.UserID != "" {
		base += fmt.Sprintf(" AND user_id = $%d", idx)
		args = append(args, *filters.UserID)
		idx++
	}
	if filters.TxType != nil && *filters.TxType != "" {
		base += fmt.Sprintf(" AND tx_type = $%d", idx)
		args = append(args, *filters.TxType)
		idx++
	}
	if filters.DateFrom != nil {
		base += fmt.Sprintf(" AND created_at >= $%d", idx)
		args = append(args, *filters.DateFrom)
		idx++
	}
	if filters.DateTo != nil {
		base += fmt.Sprintf(" AND created_at <= $%d", idx)
		args = append(args, *filters.DateTo)
		idx++
	}
	if filters.RelatedEntityType != nil && *filters.RelatedEntityType != "" {
		base += fmt.Sprintf(" AND related_entity_type = $%d", idx)
		args = append(args, *filters.RelatedEntityType)
		idx++
	}
	if filters.RelatedEntityID != nil && *filters.RelatedEntityID != "" {
		base += fmt.Sprintf(" AND related_entity_id = $%d", idx)
		args = append(args, *filters.RelatedEntityID)
		idx++
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}

	base = strings.TrimSpace(base) + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, filters.Offset)

	transactions := make([]CreditTransaction, 0)
	if err := r.db.SelectContext(ctx2, &transactions, base, args...); err != nil {
		return nil, fmt.Errorf("%w: search transactions", ErrInternal)
	}

	return transactions, nil
}

func (r *CreditRepository) insertLedger(ctx context.Context, tx *sqlx.Tx, userID string, amountDelta int, txType string, meta TxMeta) error {
	txType = strings.TrimSpace(txType)
	if txType == "" {
		txType = string(TxTypeAdminGrant)
	}

	if txType != string(TxTypeDeduction) && txType != string(TxTypeRefund) && txType != string(TxTypePurchase) && txType != string(TxTypeAdminGrant) {
		return ErrInternal
	}

	if strings.TrimSpace(meta.Description) == "" {
		meta.Description = "credit balance adjustment"
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO credit_transactions (
			id, user_id, amount_delta, tx_type, related_entity_type, related_entity_id, description
		)
		VALUES (
			gen_random_uuid(), $1, $2, $3, $4, $5, $6
		)
	`, userID, amountDelta, txType, meta.RelatedEntityType, meta.RelatedEntityID, meta.Description)
	if err != nil {
		return fmt.Errorf("%w: insert transaction", ErrInternal)
	}

	return nil
}
