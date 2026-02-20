package chat

import (
	"context"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/notification"
)

type notificationListService interface {
	List(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]*notification.Notification, error)
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
}

// NotificationQueryAdapter adapts notification service to WS sync read interface.
type NotificationQueryAdapter struct {
	svc notificationListService
}

func NewNotificationQueryAdapter(svc notificationListService) *NotificationQueryAdapter {
	return &NotificationQueryAdapter{svc: svc}
}

func (a *NotificationQueryAdapter) List(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]*notification.NotificationResponse, error) {
	items, err := a.svc.List(ctx, userID, limit, offset, unreadOnly)
	if err != nil {
		return nil, err
	}

	resp := make([]*notification.NotificationResponse, 0, len(items))
	for _, item := range items {
		resp = append(resp, notification.NotificationResponseFromEntity(item))
	}
	return resp, nil
}

func (a *NotificationQueryAdapter) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return a.svc.GetUnreadCount(ctx, userID)
}
