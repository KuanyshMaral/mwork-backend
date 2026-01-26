package payment

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Status represents payment status
type Status string

const (
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusRefunded  Status = "refunded"
)

// Provider represents payment provider
type Provider string

const (
	ProviderKaspi  Provider = "kaspi"
	ProviderCard   Provider = "card"
	ProviderManual Provider = "manual"
)

// Payment represents a payment transaction
type Payment struct {
	ID             uuid.UUID       `db:"id" json:"id"`
	UserID         uuid.UUID       `db:"user_id" json:"user_id"`
	SubscriptionID uuid.NullUUID   `db:"subscription_id" json:"subscription_id,omitempty"`
	Amount         float64         `db:"amount" json:"amount"`
	Currency       string          `db:"currency" json:"currency"`
	Status         Status          `db:"status" json:"status"`
	Provider       sql.NullString  `db:"provider" json:"provider,omitempty"`
	ExternalID     sql.NullString  `db:"external_id" json:"external_id,omitempty"`
	Description    sql.NullString  `db:"description" json:"description,omitempty"`
	Metadata       json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	PaidAt         sql.NullTime    `db:"paid_at" json:"paid_at,omitempty"`
	FailedAt       sql.NullTime    `db:"failed_at" json:"failed_at,omitempty"`
	RefundedAt     sql.NullTime    `db:"refunded_at" json:"refunded_at,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
}

// IsPaid checks if payment is completed
func (p *Payment) IsPaid() bool {
	return p.Status == StatusCompleted
}
