package review

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Review represents a model review from an employer
type Review struct {
	ID         uuid.UUID      `db:"id"`
	ProfileID  uuid.UUID      `db:"profile_id"`
	ReviewerID uuid.UUID      `db:"reviewer_id"`
	CastingID  uuid.NullUUID  `db:"casting_id"`
	Rating     int            `db:"rating"`
	Comment    sql.NullString `db:"comment"`
	IsVerified bool           `db:"is_verified"`
	IsPublic   bool           `db:"is_public"`
	CreatedAt  time.Time      `db:"created_at"`
	UpdatedAt  time.Time      `db:"updated_at"`
}

// ReviewResponse for API response
type ReviewResponse struct {
	ID           string  `json:"id"`
	ProfileID    string  `json:"profile_id"`
	ReviewerID   string  `json:"reviewer_id"`
	ReviewerName string  `json:"reviewer_name,omitempty"`
	CastingID    *string `json:"casting_id,omitempty"`
	Rating       int     `json:"rating"`
	Comment      string  `json:"comment,omitempty"`
	IsVerified   bool    `json:"is_verified"`
	CreatedAt    string  `json:"created_at"`
}

// ToResponse converts entity to response
func (r *Review) ToResponse() *ReviewResponse {
	resp := &ReviewResponse{
		ID:         r.ID.String(),
		ProfileID:  r.ProfileID.String(),
		ReviewerID: r.ReviewerID.String(),
		Rating:     r.Rating,
		IsVerified: r.IsVerified,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
	}
	if r.Comment.Valid {
		resp.Comment = r.Comment.String
	}
	if r.CastingID.Valid {
		s := r.CastingID.UUID.String()
		resp.CastingID = &s
	}
	return resp
}

// CreateRequest for creating a review
type CreateRequest struct {
	ProfileID string `json:"profile_id" validate:"required,uuid"`
	CastingID string `json:"casting_id" validate:"omitempty,uuid"`
	Rating    int    `json:"rating" validate:"required,gte=1,lte=5"`
	Comment   string `json:"comment" validate:"max=2000"`
}

// ProfileRatingSummary for profile rating overview
type ProfileRatingSummary struct {
	AverageRating float64           `json:"average_rating"`
	TotalReviews  int               `json:"total_reviews"`
	Distribution  map[int]int       `json:"distribution"`
	RecentReviews []*ReviewResponse `json:"recent_reviews,omitempty"`
}
