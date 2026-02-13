package response

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Status represents response status (matches response_status enum)
type Status string

const (
	StatusPending     Status = "pending"
	StatusViewed      Status = "viewed"
	StatusShortlisted Status = "shortlisted"
	StatusAccepted    Status = "accepted"
	StatusRejected    Status = "rejected"
)

// Response represents an application to a casting (matches casting_responses table)
type Response struct {
	ID        uuid.UUID `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	// References
	CastingID uuid.UUID `db:"casting_id"`
	ModelID   uuid.UUID `db:"model_id"` // FK to model_profiles
	UserID    uuid.UUID `db:"user_id"`

	// Application details
	Message      sql.NullString  `db:"message"`
	ProposedRate sql.NullFloat64 `db:"proposed_rate"`

	// Status
	Status     Status       `db:"status"`
	AcceptedAt sql.NullTime `db:"accepted_at"`
	RejectedAt sql.NullTime `db:"rejected_at"`

	// Review tracking
	RatingGiven bool `db:"rating_given"`

	// Joined data (not in DB)
	CastingTitle string `db:"-"`
	CastingCity  string `db:"-"`
	ModelName    string `db:"-"`
}

// IsPending returns true if response is pending
func (r *Response) IsPending() bool {
	return r.Status == StatusPending
}

// IsAccepted returns true if response is accepted
func (r *Response) IsAccepted() bool {
	return r.Status == StatusAccepted
}

// IsRejected returns true if response is rejected
func (r *Response) IsRejected() bool {
	return r.Status == StatusRejected
}

// CanBeUpdatedTo checks if status transition is valid
func (r *Response) CanBeUpdatedTo(newStatus Status) bool {
	transitions := map[Status][]Status{
		StatusPending:     {StatusViewed, StatusShortlisted, StatusAccepted, StatusRejected},
		StatusViewed:      {StatusShortlisted, StatusAccepted, StatusRejected},
		StatusShortlisted: {StatusAccepted, StatusRejected},
		StatusAccepted:    {}, // Final state
		StatusRejected:    {}, // Final state
	}

	allowed, ok := transitions[r.Status]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == newStatus {
			return true
		}
	}
	return false
}

// GetMessage returns message or empty string
func (r *Response) GetMessage() string {
	if r.Message.Valid {
		return r.Message.String
	}
	return ""
}

// GetProposedRate returns proposed rate or 0
func (r *Response) GetProposedRate() float64 {
	if r.ProposedRate.Valid {
		return r.ProposedRate.Float64
	}
	return 0
}
