package payment

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Status represents payment status
type Status string

const (
	StatusPending   Status = "pending"
	StatusPaid      Status = "paid"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusRefunded  Status = "refunded"
)

// Provider represents payment provider
type Provider string

const (
	ProviderRobokassa Provider = "robokassa"
	ProviderCard      Provider = "card"
	ProviderManual    Provider = "manual"
)

// JSONRawMessage handles NULL json fields from DB
type JSONRawMessage []byte

func (j *JSONRawMessage) Scan(src any) error {
	if src == nil {
		*j = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*j = append((*j)[0:0], v...)
	case string:
		*j = []byte(v)
	default:
		return fmt.Errorf("unsupported type: %T", src)
	}
	return nil
}

func (j JSONRawMessage) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// Payment represents a payment transaction
type Payment struct {
	ID                 uuid.UUID      `db:"id" json:"id"`
	UserID             uuid.UUID      `db:"user_id" json:"user_id"`
	PlanID             uuid.NullUUID  `db:"plan_id" json:"plan_id,omitempty"`
	SubscriptionID     uuid.NullUUID  `db:"subscription_id" json:"subscription_id,omitempty"`
	Type               string         `db:"type" json:"type,omitempty"`
	Plan               sql.NullString `db:"plan" json:"plan,omitempty"`
	InvID              sql.NullString `db:"inv_id" json:"inv_id,omitempty"`
	ResponsePackage    sql.NullInt64  `db:"response_package" json:"response_package,omitempty"`
	Amount             float64        `db:"amount" json:"amount"`
	RobokassaInvID     sql.NullInt64  `db:"robokassa_inv_id" json:"robokassa_inv_id,omitempty"`
	Currency           string         `db:"currency" json:"currency"`
	Status             Status         `db:"status" json:"status"`
	Provider           sql.NullString `db:"provider" json:"provider,omitempty"`
	ExternalID         sql.NullString `db:"external_id" json:"external_id,omitempty"`
	Description        sql.NullString `db:"description" json:"description,omitempty"`
	Metadata           JSONRawMessage `db:"metadata" json:"metadata,omitempty"`
	RawInitPayload     JSONRawMessage `db:"raw_init_payload" json:"raw_init_payload,omitempty"`
	RawCallbackPayload JSONRawMessage `db:"raw_callback_payload" json:"raw_callback_payload,omitempty"`
	PaidAt             sql.NullTime   `db:"paid_at" json:"paid_at,omitempty"`
	FailedAt           sql.NullTime   `db:"failed_at" json:"failed_at,omitempty"`
	RefundedAt         sql.NullTime   `db:"refunded_at" json:"refunded_at,omitempty"`
	CreatedAt          time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at" json:"updated_at"`
	PromotionID        uuid.NullUUID  `db:"promotion_id" json:"promotion_id,omitempty"`
}

// IsPaid checks if payment is completed
func (p *Payment) IsPaid() bool {
	return p.Status == StatusCompleted || p.Status == StatusPaid
}
