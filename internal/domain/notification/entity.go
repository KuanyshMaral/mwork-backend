package notification

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Type represents notification type
type Type string

const (
	TypeNewResponse      Type = "new_response"      // Employer: model applied
	TypeResponseAccepted Type = "response_accepted" // Model: accepted
	TypeResponseRejected Type = "response_rejected" // Model: rejected
	TypeNewMessage       Type = "new_message"       // Both: new chat message
	TypeProfileViewed    Type = "profile_viewed"    // Model: someone viewed profile (Pro)
	TypeCastingExpiring  Type = "casting_expiring"  // Employer: casting expires soon
)

// Notification represents a user notification
type Notification struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	UserID    uuid.UUID       `db:"user_id" json:"user_id"`
	Type      Type            `db:"type" json:"type"`
	Title     string          `db:"title" json:"title"`
	Body      sql.NullString  `db:"body" json:"body,omitempty"`
	Data      json.RawMessage `db:"data" json:"data,omitempty"`
	IsRead    bool            `db:"is_read" json:"is_read"`
	ReadAt    sql.NullTime    `db:"read_at" json:"read_at,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}

// NotificationData for linking to entities
type NotificationData struct {
	CastingID  *uuid.UUID `json:"casting_id,omitempty"`
	ResponseID *uuid.UUID `json:"response_id,omitempty"`
	ProfileID  *uuid.UUID `json:"profile_id,omitempty"`
	RoomID     *uuid.UUID `json:"room_id,omitempty"`
	MessageID  *uuid.UUID `json:"message_id,omitempty"`
}

// SetData encodes data to JSON
func (n *Notification) SetData(data *NotificationData) {
	if data != nil {
		n.Data, _ = json.Marshal(data)
	}
}

// GetData decodes data from JSON
func (n *Notification) GetData() *NotificationData {
	if n.Data == nil {
		return &NotificationData{}
	}
	var data NotificationData
	_ = json.Unmarshal(n.Data, &data)
	return &data
}
