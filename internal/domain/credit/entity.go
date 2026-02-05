package credit

import "time"

// TxType defines supported credit transaction types.
type TxType string

const (
	TxTypeDeduction  TxType = "deduction"
	TxTypeRefund     TxType = "refund"
	TxTypePurchase   TxType = "purchase"
	TxTypeAdminGrant TxType = "admin_grant"
)

// TxMeta represents optional metadata attached to a credit transaction.
type TxMeta struct {
	RelatedEntityType *string
	RelatedEntityID   *string
	Description       string
}

// Pagination controls simple list pagination.
type Pagination struct {
	Limit  int
	Offset int
}

// SearchFilters provides admin-facing transaction filtering.
type SearchFilters struct {
	UserID            *string
	TxType            *string
	DateFrom          *time.Time
	DateTo            *time.Time
	RelatedEntityType *string
	RelatedEntityID   *string
	Limit             int
	Offset            int
}

// CreditTransaction is a ledger row.
type CreditTransaction struct {
	ID                string    `db:"id"`
	UserID            string    `db:"user_id"`
	AmountDelta       int       `db:"amount_delta"`
	TxType            string    `db:"tx_type"`
	RelatedEntityType *string   `db:"related_entity_type"`
	RelatedEntityID   *string   `db:"related_entity_id"`
	Description       string    `db:"description"`
	CreatedAt         time.Time `db:"created_at"`
}
