package relationships

import (
	"time"

	"github.com/google/uuid"
)

// BlockUserRequest for POST /users/{id}/block
type BlockUserRequest struct {
	// No body needed - target ID is in URL
}

// UnblockUserRequest for DELETE /users/{id}/block
type UnblockUserRequest struct {
	// No body needed - target ID is in URL
}

// BlockedUserResponse represents a blocked user in API response
type BlockedUserResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	BlockedAt string    `json:"blocked_at"`
}

// BlockRelationFromEntity converts entity to response
func BlockRelationFromEntity(block *BlockRelation, firstName, lastName string, avatarURL *string) *BlockedUserResponse {
	return &BlockedUserResponse{
		ID:        block.ID,
		UserID:    block.BlockedUserID,
		FirstName: firstName,
		LastName:  lastName,
		AvatarURL: avatarURL,
		BlockedAt: block.CreatedAt.Format(time.RFC3339),
	}
}
