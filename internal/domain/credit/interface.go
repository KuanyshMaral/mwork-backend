package credit

import (
	"context"

	"github.com/google/uuid"
)

// TransactionType represents the type of credit transaction
type TransactionType string

const (
	TransactionTypeDeduction  TransactionType = "deduction"
	TransactionTypeRefund     TransactionType = "refund"
	TransactionTypePurchase   TransactionType = "purchase"
	TransactionTypeAdminGrant TransactionType = "admin_grant"
)

// TransactionMeta contains metadata for credit transactions
type TransactionMeta struct {
	RelatedEntityType string
	RelatedEntityID   uuid.UUID
	Description       string
	AdminID           *uuid.UUID // For admin grants
	PaymentID         *string    // For purchases
}

// Service interface defines the credit service operations
type Service interface {
	// Deduct atomically deducts credits from a user
	// Returns ErrInsufficientCredits if balance is insufficient
	Deduct(ctx context.Context, userID uuid.UUID, amount int, meta TransactionMeta) error

	// Add atomically adds credits to a user
	Add(ctx context.Context, userID uuid.UUID, amount int, txType TransactionType, meta TransactionMeta) error

	// GetBalance returns the current credit balance for a user
	GetBalance(ctx context.Context, userID uuid.UUID) (int, error)

	// HasRefund checks if a refund transaction already exists for a given response
	// This is used for idempotency in B2: preventing duplicate refunds on rejection
	HasRefund(ctx context.Context, responseID uuid.UUID) (bool, error)

	// ListTransactions returns paginated transaction history for a user
	ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]CreditTransaction, error)

	// SearchTransactions returns filtered transactions (for admin use)
	SearchTransactions(ctx context.Context, filters SearchFilters) ([]CreditTransaction, error)
}
