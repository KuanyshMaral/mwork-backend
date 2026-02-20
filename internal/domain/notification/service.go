package notification

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var ErrNotificationNotFound = errors.New("notification not found")

// Service handles notification logic
type Service struct {
	repo              Repository
	realtimePublisher RealtimePublisher
}

// NewService creates notification service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetRealtimePublisher sets optional realtime publisher for WS events
func (s *Service) SetRealtimePublisher(publisher RealtimePublisher) {
	s.realtimePublisher = publisher
}

// Create creates a notification
func (s *Service) Create(ctx context.Context, userID uuid.UUID, notifType Type, title, body string, data *NotificationData) (*Notification, error) {
	n := &Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	if body != "" {
		n.Body = sql.NullString{String: body, Valid: true}
	}
	n.SetData(data)

	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}

	if s.realtimePublisher != nil {
		if unreadCount, err := s.repo.CountUnreadByUser(ctx, userID); err == nil {
			_ = s.realtimePublisher.NotifyNew(ctx, userID, NotificationResponseFromEntity(n), unreadCount)
			log.Debug().Str("user_id", userID.String()).Str("notification_type", string(n.Type)).Str("source", "notification_service_create").Msg("Published notification:new WS event")
		}
	}

	return n, nil
}

// List returns notifications for user
func (s *Service) List(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]*Notification, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset, unreadOnly)
}

// GetUnreadCount returns unread count
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnreadByUser(ctx, userID)
}

// MarkAsRead marks single notification as read
func (s *Service) MarkAsRead(ctx context.Context, userID, id uuid.UUID) error {
	updated, err := s.repo.MarkAsRead(ctx, userID, id)
	if err != nil {
		return err
	}
	if !updated {
		return ErrNotificationNotFound
	}
	return nil
}

// MarkAllAsRead marks all notifications as read
func (s *Service) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	affected, err := s.repo.MarkAllAsRead(ctx, userID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

// --- Helper methods for creating specific notifications ---

// NotifyNewResponse notifies employer about new response
func (s *Service) NotifyNewResponse(ctx context.Context, employerID uuid.UUID, castingTitle string, modelName string, castingID, responseID uuid.UUID) {
	s.Create(ctx, employerID, TypeNewResponse,
		"Новый отклик на кастинг",
		modelName+" откликнулся на \""+castingTitle+"\"",
		&NotificationData{CastingID: &castingID, ResponseID: &responseID},
	)
}

// NotifyResponseAccepted notifies model about acceptance
func (s *Service) NotifyResponseAccepted(ctx context.Context, modelID uuid.UUID, castingTitle string, castingID, responseID uuid.UUID) {
	s.Create(ctx, modelID, TypeResponseAccepted,
		"Ваша заявка принята!",
		"Вас приняли на кастинг \""+castingTitle+"\"",
		&NotificationData{CastingID: &castingID, ResponseID: &responseID},
	)
}

// NotifyResponseRejected notifies model about rejection
func (s *Service) NotifyResponseRejected(ctx context.Context, modelID uuid.UUID, castingTitle string, castingID, responseID uuid.UUID) {
	s.Create(ctx, modelID, TypeResponseRejected,
		"Заявка отклонена",
		"К сожалению, ваша заявка на \""+castingTitle+"\" отклонена",
		&NotificationData{CastingID: &castingID, ResponseID: &responseID},
	)
}

// NotifyNewMessage notifies user about new message
func (s *Service) NotifyNewMessage(ctx context.Context, userID uuid.UUID, senderName, preview string, roomID, messageID uuid.UUID) {
	s.Create(ctx, userID, TypeNewMessage,
		"Новое сообщение от "+senderName,
		preview,
		&NotificationData{RoomID: &roomID, MessageID: &messageID},
	)
}
