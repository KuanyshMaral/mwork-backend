package response

import (
	"time"

	"github.com/google/uuid"
)

// ApplyRequest for POST /castings/{id}/responses
type ApplyRequest struct {
	Message      string   `json:"message" validate:"omitempty,max=2000"`
	ProposedRate *float64 `json:"proposed_rate" validate:"omitempty,gte=0"`
}

// UpdateStatusRequest for PATCH /responses/{id}/status
type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=viewed accepted rejected"`
}

// ResponseResponse represents response in API response
type ResponseResponse struct {
	ID           uuid.UUID `json:"id"`
	CastingID    uuid.UUID `json:"casting_id"`
	ModelID      uuid.UUID `json:"model_id"`
	Status       string    `json:"status"`
	Message      *string   `json:"message,omitempty"`
	ProposedRate *float64  `json:"proposed_rate,omitempty"`
	AcceptedAt   *string   `json:"accepted_at,omitempty"`
	RejectedAt   *string   `json:"rejected_at,omitempty"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`

	// Joined data
	CastingTitle string `json:"casting_title,omitempty"`
	CastingCity  string `json:"casting_city,omitempty"`
	ModelName    string `json:"model_name,omitempty"`

	// Included when listing responses for a casting
	Model *ModelSummary `json:"model,omitempty"`
}

// ModelSummary for embedding in response
type ModelSummary struct {
	ID     uuid.UUID `json:"id"`
	Name   *string   `json:"name,omitempty"`
	City   *string   `json:"city,omitempty"`
	Age    *int      `json:"age,omitempty"`
	Height *float64  `json:"height,omitempty"`
	Gender *string   `json:"gender,omitempty"`
	Rating float64   `json:"rating"`
}

// CastingSummary for embedding in response
type CastingSummary struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	City     string    `json:"city"`
	PayRange string    `json:"pay_range"`
	Status   string    `json:"status"`
}

// ResponseWithCasting includes casting info
type ResponseWithCasting struct {
	ResponseResponse
	Casting *CastingSummary `json:"casting,omitempty"`
}

// ResponseResponseFromEntity converts entity to response DTO
func ResponseResponseFromEntity(r *Response) *ResponseResponse {
	resp := &ResponseResponse{
		ID:           r.ID,
		CastingID:    r.CastingID,
		ModelID:      r.ModelID,
		Status:       string(r.Status),
		CastingTitle: r.CastingTitle,
		CastingCity:  r.CastingCity,
		ModelName:    r.ModelName,
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
	}

	if r.Message.Valid {
		resp.Message = &r.Message.String
	}
	if r.ProposedRate.Valid {
		resp.ProposedRate = &r.ProposedRate.Float64
	}
	if r.AcceptedAt.Valid {
		s := r.AcceptedAt.Time.Format(time.RFC3339)
		resp.AcceptedAt = &s
	}
	if r.RejectedAt.Valid {
		s := r.RejectedAt.Time.Format(time.RFC3339)
		resp.RejectedAt = &s
	}

	return resp
}
