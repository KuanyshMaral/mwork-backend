package notification

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/pkg/email"
	"github.com/mwork/mwork-api/internal/pkg/push"
)

// ExtendedService handles notifications with multi-channel delivery
type ExtendedService struct {
	repo         Repository
	prefsRepo    *PreferencesRepository
	deviceRepo   *DeviceTokenRepository
	emailService *email.Service
	pushClient   *push.FCMClient
	wsBroadcast  chan *WSNotification
	baseURL      string
}

// WSNotification for WebSocket broadcast
type WSNotification struct {
	UserID       uuid.UUID
	Notification *Notification
}

// ExtendedServiceConfig holds service configuration
type ExtendedServiceConfig struct {
	Repo         Repository
	PrefsRepo    *PreferencesRepository
	DeviceRepo   *DeviceTokenRepository
	EmailService *email.Service
	PushClient   *push.FCMClient
	WSBroadcast  chan *WSNotification
	BaseURL      string
}

// NewExtendedService creates extended notification service
func NewExtendedService(cfg ExtendedServiceConfig) *ExtendedService {
	return &ExtendedService{
		repo:         cfg.Repo,
		prefsRepo:    cfg.PrefsRepo,
		deviceRepo:   cfg.DeviceRepo,
		emailService: cfg.EmailService,
		pushClient:   cfg.PushClient,
		wsBroadcast:  cfg.WSBroadcast,
		baseURL:      cfg.BaseURL,
	}
}

// Send creates and delivers a notification across all enabled channels
func (s *ExtendedService) Send(ctx context.Context, params SendParams) (*Notification, error) {
	// Get user preferences
	prefs, err := s.prefsRepo.GetByUserID(ctx, params.UserID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get notification preferences, using defaults")
		prefs = &UserPreferences{
			InAppEnabled: true,
			EmailEnabled: true,
			PushEnabled:  true,
		}
	}

	channels := prefs.GetChannelsForType(params.Type)

	var notification *Notification

	// 1. In-App notification
	if channels.InApp {
		notification = &Notification{
			ID:        uuid.New(),
			UserID:    params.UserID,
			Type:      params.Type,
			Title:     params.Title,
			IsRead:    false,
			CreatedAt: time.Now(),
		}
		if params.Body != "" {
			notification.Body = sql.NullString{String: params.Body, Valid: true}
		}
		notification.SetData(params.Data)

		if err := s.repo.Create(ctx, notification); err != nil {
			log.Error().Err(err).Msg("Failed to create notification")
		} else {
			// Broadcast via WebSocket
			if s.wsBroadcast != nil {
				select {
				case s.wsBroadcast <- &WSNotification{
					UserID:       params.UserID,
					Notification: notification,
				}:
				default:
					log.Warn().Msg("WebSocket broadcast channel full")
				}
			}
		}
	}

	// 2. Email notification
	if channels.Email && s.emailService != nil && params.Email != nil {
		go s.sendEmail(params)
	}

	// 3. Push notification
	if channels.Push && s.pushClient != nil {
		go s.sendPush(ctx, params)
	}

	return notification, nil
}

// SendParams holds parameters for sending notification
type SendParams struct {
	UserID    uuid.UUID
	UserEmail string
	UserName  string
	Type      Type
	Title     string
	Body      string
	Data      *NotificationData
	Email     *EmailParams
	PushData  map[string]string
}

// EmailParams for sending email
type EmailParams struct {
	TemplateName string
	Subject      string
	TemplateData interface{}
}

// sendEmail sends email notification
func (s *ExtendedService) sendEmail(params SendParams) {
	if params.Email == nil || params.UserEmail == "" {
		return
	}

	s.emailService.Queue(
		params.UserEmail,
		params.UserName,
		params.Email.TemplateName,
		params.Email.Subject,
		params.Email.TemplateData,
	)
}

// sendPush sends push notification
func (s *ExtendedService) sendPush(ctx context.Context, params SendParams) {
	if s.deviceRepo == nil || s.pushClient == nil {
		return
	}

	tokens, err := s.deviceRepo.GetActiveByUserID(ctx, params.UserID)
	if err != nil || len(tokens) == 0 {
		return
	}

	for _, token := range tokens {
		msg := &push.PushMessage{
			Token: token,
			Title: params.Title,
			Body:  params.Body,
			Data:  params.PushData,
		}
		if err := s.pushClient.Send(context.Background(), msg); err != nil {
			log.Warn().Err(err).Str("token", token[:20]+"...").Msg("Failed to send push")
			// Deactivate invalid tokens
			if err.Error() == "FCM returned status 404" {
				s.deviceRepo.Deactivate(context.Background(), token)
			}
		}
	}
}

// --- Convenience methods ---

// NotifyResponseAccepted notifies model about acceptance with multi-channel delivery
func (s *ExtendedService) NotifyResponseAccepted(ctx context.Context, modelID uuid.UUID, modelEmail, modelName, castingTitle, employerName string, castingID, responseID uuid.UUID) {
	s.Send(ctx, SendParams{
		UserID:    modelID,
		UserEmail: modelEmail,
		UserName:  modelName,
		Type:      TypeResponseAccepted,
		Title:     "🎉 Вас приняли на кастинг!",
		Body:      "Вас приняли на \"" + castingTitle + "\"",
		Data:      &NotificationData{CastingID: &castingID, ResponseID: &responseID},
		Email: &EmailParams{
			TemplateName: "response_accepted",
			Subject:      "🎉 Вас приняли на кастинг!",
			TemplateData: map[string]string{
				"ModelName":    modelName,
				"CastingTitle": castingTitle,
				"EmployerName": employerName,
				"CastingURL":   s.baseURL + "/castings/" + castingID.String(),
			},
		},
		PushData: map[string]string{
			"type":       string(TypeResponseAccepted),
			"casting_id": castingID.String(),
		},
	})
}

// NotifyResponseRejected notifies model about rejection
func (s *ExtendedService) NotifyResponseRejected(ctx context.Context, modelID uuid.UUID, modelEmail, modelName, castingTitle string, castingID, responseID uuid.UUID) {
	s.Send(ctx, SendParams{
		UserID:    modelID,
		UserEmail: modelEmail,
		UserName:  modelName,
		Type:      TypeResponseRejected,
		Title:     "Заявка отклонена",
		Body:      "К сожалению, ваша заявка на \"" + castingTitle + "\" отклонена",
		Data:      &NotificationData{CastingID: &castingID, ResponseID: &responseID},
		Email: &EmailParams{
			TemplateName: "response_rejected",
			Subject:      "Заявка на кастинг",
			TemplateData: map[string]string{
				"CastingTitle": castingTitle,
				"CastingsURL":  s.baseURL + "/castings",
			},
		},
	})
}

// NotifyNewResponse notifies employer about new response
func (s *ExtendedService) NotifyNewResponse(ctx context.Context, employerID uuid.UUID, employerEmail, castingTitle, modelName string, castingID, responseID uuid.UUID) {
	s.Send(ctx, SendParams{
		UserID:    employerID,
		UserEmail: employerEmail,
		Type:      TypeNewResponse,
		Title:     "📩 Новый отклик на кастинг",
		Body:      modelName + " откликнулся на \"" + castingTitle + "\"",
		Data:      &NotificationData{CastingID: &castingID, ResponseID: &responseID},
		Email: &EmailParams{
			TemplateName: "new_response",
			Subject:      "📩 Новый отклик на кастинг",
			TemplateData: map[string]string{
				"CastingTitle": castingTitle,
				"ModelName":    modelName,
				"ResponseURL":  s.baseURL + "/castings/" + castingID.String() + "/responses",
			},
		},
		PushData: map[string]string{
			"type":        string(TypeNewResponse),
			"casting_id":  castingID.String(),
			"response_id": responseID.String(),
		},
	})
}

// NotifyNewMessage notifies about new chat message
func (s *ExtendedService) NotifyNewMessage(ctx context.Context, userID uuid.UUID, userEmail, senderName, preview string, roomID, messageID uuid.UUID) {
	s.Send(ctx, SendParams{
		UserID:    userID,
		UserEmail: userEmail,
		Type:      TypeNewMessage,
		Title:     "💬 Сообщение от " + senderName,
		Body:      preview,
		Data:      &NotificationData{RoomID: &roomID, MessageID: &messageID},
		Email:     nil, // Usually no email for messages
		PushData: map[string]string{
			"type":    string(TypeNewMessage),
			"room_id": roomID.String(),
		},
	})
}

// CleanupOld removes notifications older than 90 days
func (s *ExtendedService) CleanupOld(ctx context.Context) (int64, error) {
	result, err := s.repo.DeleteOlderThan(ctx, 90*24*time.Hour)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// GetPreferences gets user notification preferences
func (s *ExtendedService) GetPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	return s.prefsRepo.GetByUserID(ctx, userID)
}

// UpdatePreferences updates user notification preferences
func (s *ExtendedService) UpdatePreferences(ctx context.Context, prefs *UserPreferences) error {
	return s.prefsRepo.Update(ctx, prefs)
}

// RegisterDevice registers a device token for push notifications
func (s *ExtendedService) RegisterDevice(ctx context.Context, userID uuid.UUID, token, platform, deviceName string) error {
	return s.deviceRepo.Save(ctx, &DeviceToken{
		ID:         uuid.New(),
		UserID:     userID,
		Token:      token,
		Platform:   platform,
		DeviceName: deviceName,
		IsActive:   true,
	})
}

// UnregisterDevice unregisters a device token
func (s *ExtendedService) UnregisterDevice(ctx context.Context, token string) error {
	return s.deviceRepo.Deactivate(ctx, token)
}

// Helper to extend existing Repository interface
type DeleteOlderThanRepository interface {
	DeleteOlderThan(ctx context.Context, age time.Duration) (int64, error)
}
