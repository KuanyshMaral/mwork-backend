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

// RoomType represents the type of chat room
type RoomType string

const (
	RoomTypeDirect  RoomType = "direct"  // 1-to-1 chat
	RoomTypeCasting RoomType = "casting" // Chat linked to casting
	RoomTypeGroup   RoomType = "group"   // Multi-user group chat
)

// MemberRole represents a user's role in a chat room
type MemberRole string

const (
	MemberRoleAdmin  MemberRole = "admin"  // Room administrator
	MemberRoleMember MemberRole = "member" // Regular member
)

// Room represents a chat room
type Room struct {
	ID                 uuid.UUID      `db:"id" json:"id"`
	RoomType           RoomType       `db:"room_type" json:"room_type"`
	Name               sql.NullString `db:"name" json:"name,omitempty"`
	CreatorID          uuid.NullUUID  `db:"creator_id" json:"creator_id,omitempty"`
	CastingID          uuid.NullUUID  `db:"casting_id" json:"casting_id,omitempty"`
	LastMessageAt      sql.NullTime   `db:"last_message_at" json:"last_message_at,omitempty"`
	LastMessagePreview sql.NullString `db:"last_message_preview" json:"last_message_preview,omitempty"`
	CreatedAt          time.Time      `db:"created_at" json:"created_at"`
}

// RoomMember represents a user's membership in a chat room
type RoomMember struct {
	ID       uuid.UUID  `db:"id" json:"id"`
	RoomID   uuid.UUID  `db:"room_id" json:"room_id"`
	UserID   uuid.UUID  `db:"user_id" json:"user_id"`
	Role     MemberRole `db:"role" json:"role"`
	JoinedAt time.Time  `db:"joined_at" json:"joined_at"`
}

// IsAdmin checks if member is admin
func (m *RoomMember) IsAdmin() bool {
	return m.Role == MemberRoleAdmin
}

// Message represents a chat message
type Message struct {
	ID                 uuid.UUID     `db:"id" json:"id"`
	RoomID             uuid.UUID     `db:"room_id" json:"room_id"`
	SenderID           uuid.UUID     `db:"sender_id" json:"sender_id"`
	Content            string        `db:"content" json:"content"`
	MessageType        MessageType   `db:"message_type" json:"message_type"`
	AttachmentUploadID uuid.NullUUID `db:"attachment_upload_id" json:"attachment_upload_id,omitempty"`
	IsRead             bool          `db:"is_read" json:"is_read"`
	ReadAt             sql.NullTime  `db:"read_at" json:"read_at,omitempty"`
	CreatedAt          time.Time     `db:"created_at" json:"created_at"`
	DeletedAt          sql.NullTime  `db:"deleted_at" json:"-"`

	// ID-joined fields
	AttachmentURL  sql.NullString `db:"attachment_url" json:"-"`
	AttachmentName sql.NullString `db:"attachment_name" json:"-"`
	AttachmentMime sql.NullString `db:"attachment_mime" json:"-"`
	AttachmentSize sql.NullInt64  `db:"attachment_size" json:"-"`
}
