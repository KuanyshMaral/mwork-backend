package response

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/domain/notification"
)

// NotifiableService extends Service with notification triggers
type NotifiableService struct {
	*Service
	notifSvc *notification.ExtendedService
}

// NewNotifiableService creates a response service with notification support
func NewNotifiableService(base *Service, notifSvc *notification.ExtendedService) *NotifiableService {
	return &NotifiableService{
		Service:  base,
		notifSvc: notifSvc,
	}
}

// Apply applies to a casting and notifies the employer
func (s *NotifiableService) Apply(ctx context.Context, userID uuid.UUID, castingID uuid.UUID, req *ApplyRequest) (*Response, error) {
	// Call base Apply
	resp, err := s.Service.Apply(ctx, userID, castingID, req)
	if err != nil {
		return nil, err
	}

	// Get casting for notification
	cast, _ := s.castingRepo.GetByID(ctx, castingID)
	if cast != nil {
		// Get model profile for name
		prof, _ := s.modelRepo.GetByUserID(ctx, userID)
		modelName := "Модель"
		if prof != nil {
			modelName = prof.GetDisplayName()
		}

		// Notify employer about new response (async)
		go func() {
			s.notifSvc.NotifyNewResponse(
				context.Background(),
				cast.CreatorID,
				"", // email - will be fetched by service
				cast.Title,
				modelName,
				castingID,
				resp.ID,
			)
		}()
	}

	return resp, nil
}

// UpdateStatus updates response status and notifies the model
func (s *NotifiableService) UpdateStatus(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, newStatus Status) (*Response, error) {
	// Get response before update to track change
	resp, err := s.Service.GetByID(ctx, responseID)
	if err != nil {
		return nil, err
	}
	oldStatus := resp.Status

	// Call base UpdateStatus
	updatedResp, err := s.Service.UpdateStatus(ctx, userID, responseID, newStatus)
	if err != nil {
		return nil, err
	}

	// Only notify if status actually changed
	if oldStatus != newStatus {
		// Get casting info
		cast, _ := s.castingRepo.GetByID(ctx, updatedResp.CastingID)
		castingTitle := "кастинг"
		if cast != nil {
			castingTitle = cast.Title
		}

		// Get model's profile to get their userID
		modelProfile, _ := s.modelRepo.GetByID(ctx, updatedResp.ModelID)
		if modelProfile != nil {
			// Send notification async based on new status
			go func() {
				switch newStatus {
				case StatusAccepted:
					s.notifSvc.NotifyResponseAccepted(
						context.Background(),
						modelProfile.UserID,
						"", // email
						modelProfile.GetDisplayName(),
						castingTitle,
						"", // employer name
						updatedResp.CastingID,
						responseID,
					)
				case StatusRejected:
					s.notifSvc.NotifyResponseRejected(
						context.Background(),
						modelProfile.UserID,
						"", // email
						modelProfile.GetDisplayName(),
						castingTitle,
						updatedResp.CastingID,
						responseID,
					)
				case StatusShortlisted:
					// Use custom notification via Send for shortlist
					respID := responseID // copy for pointer
					castID := updatedResp.CastingID
					params := notification.SendParams{
						UserID: modelProfile.UserID,
						Type:   notification.TypeNewResponse, // Reuse type
						Title:  "📋 Вы в шорт-листе!",
						Body:   "Вас добавили в шорт-лист для \"" + castingTitle + "\"",
						Data: &notification.NotificationData{
							ResponseID: &respID,
							CastingID:  &castID,
						},
					}
					if _, err := s.notifSvc.Send(context.Background(), params); err != nil {
						log.Error().Err(err).Msg("Failed to send shortlist notification")
					}
				}
			}()
		}
	}

	return updatedResp, nil
}
