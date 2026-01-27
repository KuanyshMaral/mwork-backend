package chat

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// MessageType represents message type
type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeImage  MessageType = "image"
	MessageTypeSystem MessageType = "system"
)

// Room represents a chat room between two users
type Room struct {
	ID                 uuid.UUID      `db:"id" json:"id"`
	Participant1ID     uuid.UUID      `db:"participant1_id" json:"participant1_id"`
	Participant2ID     uuid.UUID      `db:"participant2_id" json:"participant2_id"`
	CastingID          uuid.NullUUID  `db:"casting_id" json:"casting_id,omitempty"`
	LastMessageAt      sql.NullTime   `db:"last_message_at" json:"last_message_at,omitempty"`
	LastMessagePreview sql.NullString `db:"last_message_preview" json:"last_message_preview,omitempty"`
	CreatedAt          time.Time      `db:"created_at" json:"created_at"`
}

// HasParticipant checks if user is in this room
func (r *Room) HasParticipant(userID uuid.UUID) bool {
	return r.Participant1ID == userID || r.Participant2ID == userID
}

// GetOtherParticipant returns the other user in the room
func (r *Room) GetOtherParticipant(userID uuid.UUID) uuid.UUID {
	if r.Participant1ID == userID {
		return r.Participant2ID
	}
	return r.Participant1ID
}

// Message represents a chat message
type Message struct {
	ID          uuid.UUID    `db:"id" json:"id"`
	RoomID      uuid.UUID    `db:"room_id" json:"room_id"`
	SenderID    uuid.UUID    `db:"sender_id" json:"sender_id"`
	Content     string       `db:"content" json:"content"`
	MessageType MessageType  `db:"message_type" json:"message_type"`
	IsRead      bool         `db:"is_read" json:"is_read"`
	ReadAt      sql.NullTime `db:"read_at" json:"read_at,omitempty"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
	DeletedAt   sql.NullTime `db:"deleted_at" json:"-"`
}
