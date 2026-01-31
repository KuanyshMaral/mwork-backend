package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/pkg/email"
	"github.com/mwork/mwork-api/internal/pkg/push"
	"github.com/rs/zerolog/log"
)

// ProfileRepository minimal interface for getting profile names
type ProfileRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (interface{}, error)
}

// IntegratedService handles notifications with email and push integration
type IntegratedService struct {
	notifService *Service
	emailService *email.Service
	pushClient   *push.FCMClient
	userRepo     user.Repository
	modelRepo    ProfileRepository
	employerRepo ProfileRepository
}

// NewIntegratedService creates an integrated notification service
func NewIntegratedService(
	notifService *Service,
	emailService *email.Service,
	pushClient *push.FCMClient,
	userRepo user.Repository,
	modelRepo ProfileRepository,
	employerRepo ProfileRepository,
) *IntegratedService {
	return &IntegratedService{
		notifService: notifService,
		emailService: emailService,
		pushClient:   pushClient,
		userRepo:     userRepo,
		modelRepo:    modelRepo,
		employerRepo: employerRepo,
	}
}

// SendWelcomeEmail sends welcome email to new user
func (s *IntegratedService) SendWelcomeEmail(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Get profile to find user's name
	displayName := "User"
	if user.Role == "model" {
		if prof, err := s.modelRepo.GetByUserID(ctx, userID); err == nil && prof != nil {
			// Type assert to get Name field
			if modelProf, ok := prof.(interface{ GetDisplayName() string }); ok {
				displayName = modelProf.GetDisplayName()
			}
		}
	} else if user.Role == "employer" {
		if prof, err := s.employerRepo.GetByUserID(ctx, userID); err == nil && prof != nil {
			// Type assert to get CompanyName field
			if empProf, ok := prof.(interface{ GetDisplayName() string }); ok {
				displayName = empProf.GetDisplayName()
			}
		}
	}

	// Send email
	s.emailService.SendWelcome(
		user.Email,
		displayName,
		displayName,
		string(user.Role),
		"https://mwork.kz/dashboard",
	)

	log.Info().
		Str("user_id", userID.String()).
		Str("email", user.Email).
		Msg("Welcome email sent")

	return nil
}

// NotifyNewResponse notifies employer about a new casting response
func (s *IntegratedService) NotifyNewResponse(ctx context.Context, employerUserID uuid.UUID, castingID uuid.UUID, responseID uuid.UUID, castingTitle string, modelName string) error {
	// Get employer user details
	employer, err := s.userRepo.GetByID(ctx, employerUserID)
	if err != nil || employer == nil {
		return fmt.Errorf("employer not found: %w", err)
	}

	// Get employer profile for display name
	employerName := "Employer"
	if prof, err := s.employerRepo.GetByUserID(ctx, employerUserID); err == nil && prof != nil {
		if empProf, ok := prof.(interface{ GetDisplayName() string }); ok {
			employerName = empProf.GetDisplayName()
		}
	}

	// Create in-app notification
	_, err = s.notifService.Create(
		ctx,
		employerUserID,
		TypeNewResponse,
		"Новый отклик на кастинг",
		fmt.Sprintf("%s откликнулся на \"%s\"", modelName, castingTitle),
		&NotificationData{
			CastingID:  &castingID,
			ResponseID: &responseID,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create in-app notification")
	}

	// Send email notification
	responseURL := fmt.Sprintf("https://mwork.kz/castings/%s/responses/%s", castingID.String(), responseID.String())
	s.emailService.SendNewResponse(
		employer.Email,
		employerName,
		castingTitle,
		modelName,
		responseURL,
	)

	// Send push notification if user has FCM token
	// TODO: Get user's FCM tokens from database
	// For now, we'll just log
	log.Info().
		Str("employer_id", employerUserID.String()).
		Str("casting_id", castingID.String()).
		Msg("New response notification sent")

	return nil
}

// NotifyAgencyFollowersNewCasting notifies all followers of an organization about a new casting
func (s *IntegratedService) NotifyAgencyFollowersNewCasting(ctx context.Context, organizationID uuid.UUID, castingID uuid.UUID, castingTitle string) error {
	// This will be wired when casting service is updated
	// For now, we'll create a placeholder that can be called

	log.Info().
		Str("organization_id", organizationID.String()).
		Str("casting_id", castingID.String()).
		Str("title", castingTitle).
		Msg("Agency followers notification triggered")

	// TODO:
	// 1. Get all followers from organization repository
	// 2. For each follower, create in-app notification
	// 3. Send push notification to each follower
	// 4. Optionally send email digest

	return nil
}

// NotifyResponseStatusChange notifies model when their response status changes
func (s *IntegratedService) NotifyResponseStatusChange(ctx context.Context, modelUserID uuid.UUID, castingTitle string, status string, castingID uuid.UUID, responseID uuid.UUID) error {
	model, err := s.userRepo.GetByID(ctx, modelUserID)
	if err != nil || model == nil {
		return fmt.Errorf("model not found: %w", err)
	}

	// Get model profile for display name
	modelName := "Model"
	if prof, err := s.modelRepo.GetByUserID(ctx, modelUserID); err == nil && prof != nil {
		if modelProf, ok := prof.(interface{ GetDisplayName() string }); ok {
			modelName = modelProf.GetDisplayName()
		}
	}

	var notifType Type
	var title, body string

	switch status {
	case "accepted":
		notifType = TypeResponseAccepted
		title = "Ваша заявка принята!"
		body = fmt.Sprintf("Вас приняли на кастинг \"%s\"", castingTitle)

		// Send email
		castingURL := fmt.Sprintf("https://mwork.kz/castings/%s", castingID.String())
		s.emailService.SendResponseAccepted(
			model.Email,
			modelName,
			modelName,
			castingTitle,
			"Работодатель", // TODO: Get actual employer name
			castingURL,
		)

	case "rejected":
		notifType = TypeResponseRejected
		title = "Заявка отклонена"
		body = fmt.Sprintf("К сожалению, ваша заявка на \"%s\" отклонена", castingTitle)

		// Send email
		s.emailService.SendResponseRejected(
			model.Email,
			modelName,
			castingTitle,
			"https://mwork.kz/castings",
		)

	default:
		return nil // Unknown status, skip notification
	}

	// Create in-app notification
	_, err = s.notifService.Create(
		ctx,
		modelUserID,
		notifType,
		title,
		body,
		&NotificationData{
			CastingID:  &castingID,
			ResponseID: &responseID,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create in-app notification")
	}

	log.Info().
		Str("model_id", modelUserID.String()).
		Str("status", status).
		Msg("Response status change notification sent")

	return nil
}

// NotifyNewMessage notifies user about a new chat message
func (s *IntegratedService) NotifyNewMessage(ctx context.Context, recipientUserID uuid.UUID, senderName string, messagePreview string, roomID uuid.UUID, messageID uuid.UUID) error {
	recipient, err := s.userRepo.GetByID(ctx, recipientUserID)
	if err != nil || recipient == nil {
		return fmt.Errorf("recipient not found: %w", err)
	}

	// Get recipient profile for display name
	recipientName := "User"
	if recipient.Role == "model" {
		if prof, err := s.modelRepo.GetByUserID(ctx, recipientUserID); err == nil && prof != nil {
			if modelProf, ok := prof.(interface{ GetDisplayName() string }); ok {
				recipientName = modelProf.GetDisplayName()
			}
		}
	} else if recipient.Role == "employer" {
		if prof, err := s.employerRepo.GetByUserID(ctx, recipientUserID); err == nil && prof != nil {
			if empProf, ok := prof.(interface{ GetDisplayName() string }); ok {
				recipientName = empProf.GetDisplayName()
			}
		}
	}

	// Create in-app notification
	_, err = s.notifService.Create(
		ctx,
		recipientUserID,
		TypeNewMessage,
		fmt.Sprintf("Новое сообщение от %s", senderName),
		messagePreview,
		&NotificationData{
			RoomID:    &roomID,
			MessageID: &messageID,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create in-app notification")
	}

	// Send email notification
	chatURL := fmt.Sprintf("https://mwork.kz/chat/%s", roomID.String())
	s.emailService.SendNewMessage(
		recipient.Email,
		recipientName,
		senderName,
		messagePreview,
		chatURL,
	)

	log.Info().
		Str("recipient_id", recipientUserID.String()).
		Msg("New message notification sent")

	return nil
}
