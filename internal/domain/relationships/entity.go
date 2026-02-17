package relationships

import (
	"time"

	"github.com/google/uuid"
)

// BlockRelation represents a user-to-user block
type BlockRelation struct {
	ID            uuid.UUID `db:"id" json:"id"`
	BlockerUserID uuid.UUID `db:"blocker_user_id" json:"blocker_user_id"`
	BlockedUserID uuid.UUID `db:"blocked_user_id" json:"blocked_user_id"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}
