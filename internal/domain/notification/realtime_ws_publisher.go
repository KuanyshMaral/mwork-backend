package notification

import (
	"context"

	"github.com/google/uuid"
)

type wsUserSender interface {
	SendToUserJSON(userID uuid.UUID, payload any) error
}

// WSPublisher publishes notification:new events over websocket.
type WSPublisher struct {
	sender wsUserSender
}

// NewWSPublisher creates a WS-backed realtime publisher.
func NewWSPublisher(sender wsUserSender) *WSPublisher {
	return &WSPublisher{sender: sender}
}

func (p *WSPublisher) NotifyNew(ctx context.Context, userID uuid.UUID, notification *NotificationResponse, unreadCount int) error {
	if p == nil || p.sender == nil {
		return nil
	}

	payload := map[string]interface{}{
		"type": "notification:new",
		"data": map[string]interface{}{
			"notification": notification,
			"unread_count": unreadCount,
		},
	}

	return p.sender.SendToUserJSON(userID, payload)
}
