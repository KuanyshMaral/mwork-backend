package review

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// TargetType represents what kind of entity is being reviewed.
type TargetType string

const (
	TargetTypeModelProfile    TargetType = "model_profile"
	TargetTypeEmployerProfile TargetType = "employer_profile"
	TargetTypeCasting         TargetType = "casting"
)

// Review represents a polymorphic review in the system.
type Review struct {
	ID          uuid.UUID      `db:"id"`
	AuthorID    uuid.UUID      `db:"author_id"`
	AuthorName  sql.NullString `db:"author_name"`
	TargetType  TargetType     `db:"target_type"`
	TargetID    uuid.UUID      `db:"target_id"`
	ContextType sql.NullString `db:"context_type"`
	ContextID   uuid.NullUUID  `db:"context_id"`
	Rating      int            `db:"rating"`
	Comment     sql.NullString `db:"comment"`
	Criteria    []byte         `db:"criteria"` // raw JSONB
	IsVerified  bool           `db:"is_verified"`
	IsPublic    bool           `db:"is_public"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

// ReviewResponse is the API response shape.
type ReviewResponse struct {
	ID          string         `json:"id"`
	AuthorID    string         `json:"author_id"`
	AuthorName  string         `json:"author_name,omitempty"`
	TargetType  string         `json:"target_type"`
	TargetID    string         `json:"target_id"`
	ContextType *string        `json:"context_type,omitempty"`
	ContextID   *string        `json:"context_id,omitempty"`
	Rating      int            `json:"rating"`
	Comment     string         `json:"comment,omitempty"`
	Criteria    map[string]int `json:"criteria,omitempty"`
	IsVerified  bool           `json:"is_verified"`
	CreatedAt   string         `json:"created_at"`
}

// ToResponse converts entity to API response.
func (r *Review) ToResponse() *ReviewResponse {
	resp := &ReviewResponse{
		ID:         r.ID.String(),
		AuthorID:   r.AuthorID.String(),
		TargetType: string(r.TargetType),
		TargetID:   r.TargetID.String(),
		Rating:     r.Rating,
		IsVerified: r.IsVerified,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
	}
	if r.AuthorName.Valid {
		resp.AuthorName = r.AuthorName.String
	}
	if r.Comment.Valid {
		resp.Comment = r.Comment.String
	}
	if r.ContextType.Valid {
		v := r.ContextType.String
		resp.ContextType = &v
	}
	if r.ContextID.Valid {
		s := r.ContextID.UUID.String()
		resp.ContextID = &s
	}
	return resp
}

// CreateRequest for creating a review.
type CreateRequest struct {
	TargetType  string         `json:"target_type" validate:"required,oneof=model_profile employer_profile casting"`
	TargetID    string         `json:"target_id" validate:"required,uuid"`
	ContextType string         `json:"context_type" validate:"omitempty,oneof=casting"`
	ContextID   string         `json:"context_id" validate:"omitempty,uuid"`
	Rating      int            `json:"rating" validate:"required,gte=1,lte=5"`
	Comment     string         `json:"comment" validate:"max=2000"`
	Criteria    map[string]int `json:"criteria"`
}

// TargetRatingSummary holds aggregated stats for any target entity.
type TargetRatingSummary struct {
	AverageRating float64           `json:"average_rating"`
	TotalReviews  int               `json:"total_reviews"`
	Distribution  map[int]int       `json:"distribution"`
	RecentReviews []*ReviewResponse `json:"recent_reviews,omitempty"`
}
