package notification

import (
	"context"

	"github.com/google/uuid"
)

// RealtimePublisher publishes in-app notification realtime events.
type RealtimePublisher interface {
	NotifyNew(ctx context.Context, userID uuid.UUID, notification *NotificationResponse, unreadCount int) error
}
