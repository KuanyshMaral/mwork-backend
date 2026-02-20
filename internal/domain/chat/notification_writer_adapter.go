package chat

import (
	"context"

	"github.com/google/uuid"
)

type notificationWriteService interface {
	MarkAsRead(ctx context.Context, userID, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
}

// NotificationWriterAdapter adapts notification service write methods for WS commands.
type NotificationWriterAdapter struct {
	svc notificationWriteService
}

func NewNotificationWriterAdapter(svc notificationWriteService) *NotificationWriterAdapter {
	return &NotificationWriterAdapter{svc: svc}
}

func (a *NotificationWriterAdapter) MarkAsRead(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	return a.svc.MarkAsRead(ctx, userID, id)
}

func (a *NotificationWriterAdapter) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	return a.svc.MarkAllAsRead(ctx, userID)
}

func (a *NotificationWriterAdapter) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return a.svc.GetUnreadCount(ctx, userID)
}
