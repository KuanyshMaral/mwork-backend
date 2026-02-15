package wallet

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) EnsureWallet(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_wallets (user_id, balance)
		VALUES ($1, 0)
		ON CONFLICT (user_id) DO NOTHING
	`, userID)
	return err
}

func (r *Repository) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	if err := r.EnsureWallet(ctx, userID); err != nil {
		return 0, err
	}

	var balance int64
	err := r.db.GetContext(ctx, &balance, `SELECT balance FROM user_wallets WHERE user_id = $1`, userID)
	return balance, err
}

func (r *Repository) beginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
}

func (r *Repository) lockWallet(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID) (int64, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_wallets (user_id, balance)
		VALUES ($1, 0)
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return 0, err
	}

	var balance int64
	err := tx.GetContext(ctx, &balance, `SELECT balance FROM user_wallets WHERE user_id = $1 FOR UPDATE`, userID)
	return balance, err
}

func (r *Repository) getTransactionAmountByRef(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, txType TransactionType, referenceID string) (int64, bool, error) {
	if referenceID == "" {
		return 0, false, nil
	}

	var amount int64
	err := tx.GetContext(ctx, &amount, `
		SELECT amount
		FROM wallet_transactions
		WHERE user_id = $1 AND type = $2 AND reference_id = $3
		LIMIT 1
	`, userID, string(txType), referenceID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return amount, true, nil
}

func (r *Repository) updateBalance(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, balance int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE user_wallets SET balance = $1, updated_at = now() WHERE user_id = $2`, balance, userID)
	return err
}

func (r *Repository) insertTransaction(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, amount int64, txType TransactionType, referenceID string) error {
	var ref interface{}
	if referenceID == "" {
		ref = nil
	} else {
		ref = referenceID
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions (user_id, amount, type, reference_id)
		VALUES ($1, $2, $3, $4)
	`, userID, amount, string(txType), ref)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrDuplicateReference
		}
		return err
	}
	return nil
}

func (r *Repository) apply(ctx context.Context, userID uuid.UUID, amount int64, txType TransactionType, referenceID string) error {
	tx, err := r.beginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	balance, err := r.lockWallet(ctx, tx, userID)
	if err != nil {
		return err
	}

	existingAmount, exists, err := r.getTransactionAmountByRef(ctx, tx, userID, txType, referenceID)
	if err != nil {
		return err
	}
	if exists {
		if existingAmount != amount {
			return ErrReferenceConflict
		}
		return nil
	}

	nextBalance := balance + amount
	if nextBalance < 0 {
		return ErrInsufficientFunds
	}

	if err := r.updateBalance(ctx, tx, userID, nextBalance); err != nil {
		return err
	}

	if err := r.insertTransaction(ctx, tx, userID, amount, txType, referenceID); err != nil {
		if errors.Is(err, ErrDuplicateReference) {
			existingAmount, exists, checkErr := r.getTransactionAmountByRef(ctx, tx, userID, txType, referenceID)
			if checkErr != nil {
				return checkErr
			}
			if !exists || existingAmount != amount {
				return ErrReferenceConflict
			}
			return nil
		}
		return err
	}

	return tx.Commit()
}

func (r *Repository) TopUp(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	return r.apply(ctx, userID, amount, TransactionTypeTopUp, referenceID)
}

func (r *Repository) Spend(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	return r.apply(ctx, userID, -amount, TransactionTypePayment, referenceID)
}

func (r *Repository) Refund(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	return r.apply(ctx, userID, amount, TransactionTypeRefund, referenceID)
}
