package chat

import (
	"time"

	"github.com/google/uuid"
)

// CreateRoomRequest for POST /chat/rooms
type CreateRoomRequest struct {
	RoomType    string      `json:"room_type" validate:"required,oneof=direct casting group"`
	RecipientID *uuid.UUID  `json:"recipient_id,omitempty"` // For direct and casting rooms
	MemberIDs   []uuid.UUID `json:"member_ids,omitempty"`   // For group rooms
	Name        string      `json:"name,omitempty"`         // For group rooms
	CastingID   *uuid.UUID  `json:"casting_id,omitempty"`   // For casting rooms
	Message     string      `json:"message,omitempty"`      // Optional initial message
}

// SendMessageRequest for WebSocket/POST
type SendMessageRequest struct {
	Content             string      `json:"content,omitempty"`
	MessageType         string      `json:"message_type,omitempty"`
	AttachmentUploadIDs []uuid.UUID `json:"attachment_upload_ids,omitempty"`
}

// MarkReadRequest for POST /chat/rooms/{id}/read
type MarkReadRequest struct {
	LastReadMessageID *uuid.UUID `json:"last_read_message_id,omitempty"`
}

// RoomResponse represents room in API
type RoomResponse struct {
	ID                 uuid.UUID         `json:"id"`
	RoomType           string            `json:"room_type"`
	Name               *string           `json:"name,omitempty"`
	Members            []ParticipantInfo `json:"members"`
	IsAdmin            bool              `json:"is_admin"`
	CastingID          *uuid.UUID        `json:"casting_id,omitempty"`
	LastMessageAt      *string           `json:"last_message_at,omitempty"`
	LastMessagePreview *string           `json:"last_message_preview,omitempty"`
	UnreadCount        int               `json:"unread_count"`
	CreatedAt          string            `json:"created_at"`
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
	ID          uuid.UUID         `json:"id"`
	RoomID      uuid.UUID         `json:"room_id"`
	SenderID    uuid.UUID         `json:"sender_id"`
	Content     string            `json:"content"`
	MessageType string            `json:"message_type"`
	Attachments []*AttachmentInfo `json:"attachments,omitempty"`
	IsRead      bool              `json:"is_read"`
	IsMine      bool              `json:"is_mine"` // Helper for client
	CreatedAt   string            `json:"created_at"`
}

// MessageResponseFromEntity converts entity to response
func MessageResponseFromEntity(m *Message, currentUserID uuid.UUID) *MessageResponse {
	resp := &MessageResponse{
		ID:          m.ID,
		RoomID:      m.RoomID,
		SenderID:    m.SenderID,
		Content:     m.Content,
		MessageType: string(m.MessageType),
		IsRead:      m.IsRead,
		IsMine:      m.SenderID == currentUserID,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
	}

	if len(m.Attachments) > 0 {
		resp.Attachments = m.Attachments
	}

	return resp
}

// RoomResponseFromEntity will be updated in service layer to populate members
// This is just a placeholder - actual implementation will fetch members from repository
func RoomResponseFromEntity(r *Room, members []ParticipantInfo, isAdmin bool, unreadCount int) *RoomResponse {
	resp := &RoomResponse{
		ID:          r.ID,
		RoomType:    string(r.RoomType),
		Members:     members,
		IsAdmin:     isAdmin,
		UnreadCount: unreadCount,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
	}

	if r.Name.Valid {
		resp.Name = &r.Name.String
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

// AddMemberRequest for POST /chat/rooms/{id}/members
type AddMemberRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
}

// RemoveMemberRequest for DELETE /chat/rooms/{id}/members/{userId}
// No body needed, userId in URL

type SendMessageRequestDoc struct {
	Text                string      `json:"text,omitempty" example:"optional text"`
	AttachmentUploadIDs []uuid.UUID `json:"attachment_upload_ids,omitempty" example:"[\"550e8400-e29b-41d4-a716-446655440000\"]"`
}
