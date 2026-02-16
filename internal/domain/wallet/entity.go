package wallet

import (
	"time"

	"github.com/google/uuid"
)

type TransactionType string

const (
	TransactionTypeTopUp   TransactionType = "topup"
	TransactionTypePayment TransactionType = "payment"
	TransactionTypeRefund  TransactionType = "refund"
)

type Wallet struct {
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	Balance   int64     `db:"balance" json:"balance"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Transaction struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	UserID      uuid.UUID       `db:"user_id" json:"user_id"`
	Amount      int64           `db:"amount" json:"amount"`
	Type        TransactionType `db:"type" json:"type"`
	ReferenceID *string         `db:"reference_id" json:"reference_id,omitempty"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
}
