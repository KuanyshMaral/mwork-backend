package chat

import (
	"context"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/notification"
)

// NotifiableService extends Service with notification triggers
type NotifiableService struct {
	*Service
	notifSvc *notification.ExtendedService
}

// NewNotifiableService creates a chat service with notification support
func NewNotifiableService(base *Service, notifSvc *notification.ExtendedService) *NotifiableService {
	return &NotifiableService{
		Service:  base,
		notifSvc: notifSvc,
	}
}

// SendMessage sends a message and notifies the recipient
func (s *NotifiableService) SendMessage(ctx context.Context, userID, roomID uuid.UUID, req *SendMessageRequest) (*Message, error) {
	// Call base SendMessage
	msg, err := s.Service.SendMessage(ctx, userID, roomID, req)
	if err != nil {
		return nil, err
	}

	// Get members to find recipients
	members, err := s.repo.GetMembers(ctx, roomID)
	if err == nil {
		// Get sender info
		sender, _ := s.userRepo.GetByID(ctx, userID)
		senderName := "Пользователь"
		if sender != nil {
			senderName = sender.Email
		}

		// Preview of message
		preview := req.Content
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		if req.MessageType == "image" {
			preview = "📷 Фото"
		}
		if req.AttachmentUploadID != nil {
			preview = "📎 Вложение"
		}

		// Send notification to all other members
		for _, member := range members {
			if member.UserID == userID {
				continue
			}

			recipientID := member.UserID
			// Send notification async
			go func(rid uuid.UUID) {
				s.notifSvc.NotifyNewMessage(
					context.Background(),
					rid,
					"", // email
					senderName,
					preview,
					roomID,
					msg.ID,
				)
			}(recipientID)
		}
	}

	return msg, nil
}
