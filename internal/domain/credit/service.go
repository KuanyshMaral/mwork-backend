package credit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// service implements the Service interface
type service struct {
	repo *CreditRepository
}

// NewService creates a new credit service
func NewService(db *sqlx.DB) Service {
	return &service{
		repo: NewRepository(db),
	}
}

// Deduct atomically deducts credits from a user
// B1: Used when applying to castings - deducts 1 credit before creating response
func (s *service) Deduct(ctx context.Context, userID uuid.UUID, amount int, meta TransactionMeta) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	txMeta := TxMeta{
		Description: meta.Description,
	}

	if meta.RelatedEntityType != "" {
		txMeta.RelatedEntityType = &meta.RelatedEntityType
	}

	if meta.RelatedEntityID != uuid.Nil {
		entityIDStr := meta.RelatedEntityID.String()
		txMeta.RelatedEntityID = &entityIDStr
	}

	return s.repo.Deduct(ctx, userID.String(), amount, txMeta)
}

// DeductTx deducts credits within an external transaction (FOR UPDATE row lock).
// Used when credit deduction must be atomic with another operation (e.g. creating a response).
func (s *service) DeductTx(ctx context.Context, tx *sqlx.Tx, userID uuid.UUID, amount int, meta TransactionMeta) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	txMeta := TxMeta{
		Description: meta.Description,
	}

	if meta.RelatedEntityType != "" {
		txMeta.RelatedEntityType = &meta.RelatedEntityType
	}

	if meta.RelatedEntityID != uuid.Nil {
		entityIDStr := meta.RelatedEntityID.String()
		txMeta.RelatedEntityID = &entityIDStr
	}

	return s.repo.DeductTx(ctx, tx, userID.String(), amount, txMeta)
}

// Add atomically adds credits to a user
// B2: Used for refunds when casting is rejected
// B3: Used by admin to grant credits
// B4: Used when user purchases credits
func (s *service) Add(ctx context.Context, userID uuid.UUID, amount int, txType TransactionType, meta TransactionMeta) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	// Convert TransactionMeta to TxMeta for repository
	txMeta := TxMeta{
		Description: meta.Description,
	}

	if meta.RelatedEntityType != "" {
		txMeta.RelatedEntityType = &meta.RelatedEntityType
	}

	if meta.RelatedEntityID != uuid.Nil {
		entityIDStr := meta.RelatedEntityID.String()
		txMeta.RelatedEntityID = &entityIDStr
	}

	return s.repo.Add(ctx, userID.String(), amount, string(txType), txMeta)
}

// GetBalance returns the current credit balance for a user
func (s *service) GetBalance(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.GetBalance(ctx, userID.String())
}

// HasRefund checks if a refund transaction already exists for a given response
// B2: Used to ensure idempotency - prevents duplicate refunds on rejection
func (s *service) HasRefund(ctx context.Context, responseID uuid.UUID) (bool, error) {
	// Check if there's already a refund transaction for this response
	entityType := "response"
	entityIDStr := responseID.String()

	filters := SearchFilters{
		TxType:            ptrString(string(TransactionTypeRefund)),
		RelatedEntityType: &entityType,
		RelatedEntityID:   &entityIDStr,
		Limit:             1,
		Offset:            0,
	}

	transactions, err := s.repo.SearchTransactions(ctx, filters)
	if err != nil {
		// If there's an error checking, treat as no refund to be safe
		// This will be logged by the caller
		return false, fmt.Errorf("failed to check refund existence: %w", err)
	}

	return len(transactions) > 0, nil
}

// Helper function to create string pointer
func ptrString(s string) *string {
	return &s
}

// ListTransactions returns paginated transaction history for a user
func (s *service) ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]CreditTransaction, error) {
	if limit <= 0 {
		limit = 20
	}

	pagination := Pagination{
		Limit:  limit,
		Offset: offset,
	}

	return s.repo.ListTransactions(ctx, userID.String(), pagination)
}

// SearchTransactions returns filtered transactions (admin use)
func (s *service) SearchTransactions(ctx context.Context, filters SearchFilters) ([]CreditTransaction, error) {
	return s.repo.SearchTransactions(ctx, filters)
}
