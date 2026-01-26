package notification

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Service handles notification logic
type Service struct {
	repo Repository
}

// NewService creates notification service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

	return n, nil
}

// List returns notifications for user
func (s *Service) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

// GetUnreadCount returns unread count
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnreadByUser(ctx, userID)
}

// MarkAsRead marks single notification as read
func (s *Service) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	return s.repo.MarkAsRead(ctx, id)
}

// MarkAllAsRead marks all notifications as read
func (s *Service) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return s.repo.MarkAllAsRead(ctx, userID)
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
