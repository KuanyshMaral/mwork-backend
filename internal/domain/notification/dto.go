package notification

import (
	"time"

	"github.com/google/uuid"
)

// NotificationResponse for API
type NotificationResponse struct {
	ID        uuid.UUID         `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Body      *string           `json:"body,omitempty"`
	Data      *NotificationData `json:"data,omitempty"`
	IsRead    bool              `json:"is_read"`
	CreatedAt string            `json:"created_at"`
}

// NotificationResponseFromEntity converts entity to response
func NotificationResponseFromEntity(n *Notification) *NotificationResponse {
	resp := &NotificationResponse{
		ID:        n.ID,
		Type:      string(n.Type),
		Title:     n.Title,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt.Format(time.RFC3339),
	}

	if n.Body.Valid {
		resp.Body = &n.Body.String
	}

	if n.Data != nil && len(n.Data) > 0 {
		resp.Data = n.GetData()
	}

	return resp
}

// UnreadCountResponse for unread count endpoint
type UnreadCountResponse struct {
	UnreadCount int `json:"unread_count"`
}
