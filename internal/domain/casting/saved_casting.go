package casting

import (
	"time"

	"github.com/google/uuid"
)

// SavedCasting represents a user's saved/bookmarked casting
type SavedCasting struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	CastingID uuid.UUID `db:"casting_id"`
	CreatedAt time.Time `db:"created_at"`
}

// SavedCastingResponse for API response
type SavedCastingResponse struct {
	CastingID string `json:"casting_id"`
	SavedAt   string `json:"saved_at"`
}

// ToResponse converts entity to response
func (s *SavedCasting) ToResponse() SavedCastingResponse {
	return SavedCastingResponse{
		CastingID: s.CastingID.String(),
		SavedAt:   s.CreatedAt.Format(time.RFC3339),
	}
}
