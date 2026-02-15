package chat

import (
	"time"

	"github.com/google/uuid"
)

// CreateRoomRequest for POST /chat/rooms
type CreateRoomRequest struct {
	RecipientID uuid.UUID  `json:"recipient_id" validate:"required"`
	CastingID   *uuid.UUID `json:"casting_id,omitempty"`
	Message     string     `json:"message,omitempty"` // Optional initial message
}

// SendMessageRequest for WebSocket/POST
type SendMessageRequest struct {
	Content     string `json:"content" validate:"required,min=1,max=5000"`
	MessageType string `json:"message_type,omitempty"`
}

// MarkReadRequest for POST /chat/rooms/{id}/read
type MarkReadRequest struct {
	LastReadMessageID *uuid.UUID `json:"last_read_message_id,omitempty"`
}

// RoomResponse represents room in API
type RoomResponse struct {
	ID                 uuid.UUID        `json:"id"`
	OtherParticipantID uuid.UUID        `json:"other_participant_id"`
	OtherParticipant   *ParticipantInfo `json:"other_participant,omitempty"`
	CastingID          *uuid.UUID       `json:"casting_id,omitempty"`
	LastMessageAt      *string          `json:"last_message_at,omitempty"`
	LastMessagePreview *string          `json:"last_message_preview,omitempty"`
	UnreadCount        int              `json:"unread_count"`
	CreatedAt          string           `json:"created_at"`
}

// ParticipantInfo for room response
type ParticipantInfo struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
}

// MessageResponse represents message in API
type MessageResponse struct {
	ID          uuid.UUID `json:"id"`
	RoomID      uuid.UUID `json:"room_id"`
	SenderID    uuid.UUID `json:"sender_id"`
	Content     string    `json:"content"`
	MessageType string    `json:"message_type"`
	IsRead      bool      `json:"is_read"`
	IsMine      bool      `json:"is_mine"` // Helper for client
	CreatedAt   string    `json:"created_at"`
}

// MessageResponseFromEntity converts entity to response
func MessageResponseFromEntity(m *Message, currentUserID uuid.UUID) *MessageResponse {
	return &MessageResponse{
		ID:          m.ID,
		RoomID:      m.RoomID,
		SenderID:    m.SenderID,
		Content:     m.Content,
		MessageType: string(m.MessageType),
		IsRead:      m.IsRead,
		IsMine:      m.SenderID == currentUserID,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
	}
}

// RoomResponseFromEntity converts entity to response
func RoomResponseFromEntity(r *Room, currentUserID uuid.UUID, unreadCount int) *RoomResponse {
	resp := &RoomResponse{
		ID:                 r.ID,
		OtherParticipantID: r.GetOtherParticipant(currentUserID),
		UnreadCount:        unreadCount,
		CreatedAt:          r.CreatedAt.Format(time.RFC3339),
	}

	if r.CastingID.Valid {
		resp.CastingID = &r.CastingID.UUID
	}
	if r.LastMessageAt.Valid {
		s := r.LastMessageAt.Time.Format(time.RFC3339)
		resp.LastMessageAt = &s
	}
	if r.LastMessagePreview.Valid {
		resp.LastMessagePreview = &r.LastMessagePreview.String
	}

	return resp
}

type SendMessageRequestDoc struct {
	Text               string `json:"text,omitempty" example:"optional text"`
	AttachmentUploadID string `json:"attachment_upload_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
}
