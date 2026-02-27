package chat

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

// AccessChecker defines interface for checking communication access between users
type AccessChecker interface {
	CanCommunicate(ctx context.Context, user1, user2 uuid.UUID) error
}

// LimitChecker defines interface for subscription limit checks.
type LimitChecker interface {
	CanUseChat(ctx context.Context, userID uuid.UUID) error
}

// UploadResolver defines interface for resolving upload details
type UploadResolver interface {
	GetAttachmentInfo(ctx context.Context, uploadID, userID uuid.UUID) (*AttachmentInfo, error)
}

// NotificationService defines notification operations used by chat service
type NotificationService interface {
	NotifyNewMessage(ctx context.Context, recipientUserID uuid.UUID, senderName string, messagePreview string, roomID uuid.UUID, messageID uuid.UUID) error
}

// Service handles chat business logic
type Service struct {
	repo           Repository
	userRepo       user.Repository
	hub            *Hub // WebSocket hub
	accessChecker  AccessChecker
	limitChecker   LimitChecker
	uploadResolver UploadResolver
	notifService   NotificationService
}

// NewService creates chat service
func NewService(repo Repository, userRepo user.Repository, hub *Hub, accessChecker AccessChecker, limitChecker LimitChecker, uploadResolver UploadResolver) *Service {
	return &Service{
		repo:           repo,
		userRepo:       userRepo,
		hub:            hub,
		accessChecker:  accessChecker,
		limitChecker:   limitChecker,
		uploadResolver: uploadResolver,
	}
}

// SetNotificationService sets optional notification service for new message notifications
func (s *Service) SetNotificationService(notifService NotificationService) {
	s.notifService = notifService
}

// CreateOrGetRoom creates a room or returns existing one (router method)
func (s *Service) CreateOrGetRoom(ctx context.Context, userID uuid.UUID, req *CreateRoomRequest) (*Room, error) {
	switch RoomType(req.RoomType) {
	case RoomTypeDirect:
		if req.RecipientID == nil {
			return nil, ErrUserNotFound
		}
		room, err := s.CreateDirectRoom(ctx, userID, *req.RecipientID, req.CastingID)
		if err != nil {
			return nil, err
		}
		// Send initial message if provided
		if req.Message != "" {
			_, _ = s.SendMessage(ctx, userID, room.ID, &SendMessageRequest{
				Content:     req.Message,
				MessageType: "text",
			})
		}
		return room, nil

	case RoomTypeCasting:
		if req.CastingID == nil {
			return nil, ErrCastingRequired
		}
		if req.RecipientID == nil {
			return nil, ErrUserNotFound
		}

		// Variant A: Unified Direct Chat
		// Map casting chat request to a direct chat between the employer and model
		room, err := s.CreateDirectRoom(ctx, userID, *req.RecipientID, req.CastingID)
		if err != nil {
			return nil, err
		}

		// Send system message to explicitly indicate casting context
		msgContent := req.Message
		msgType := "text"
		if msgContent == "" {
			msgContent = "Отклик на кастинг" // Generic context if no explicit message was provided
			msgType = "system"
		}

		_, _ = s.SendMessage(ctx, userID, room.ID, &SendMessageRequest{
			Content:     msgContent,
			MessageType: msgType,
		})

		return room, nil

	case RoomTypeGroup:
		return s.CreateGroupRoom(ctx, userID, req.MemberIDs, req.Name)

	default:
		return nil, ErrInvalidRoomType
	}
}

// GetRoom returns room by ID
func (s *Service) GetRoom(ctx context.Context, userID, roomID uuid.UUID) (*Room, error) {
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil {
		return nil, ErrRoomNotFound
	}

	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotRoomMember
	}

	return room, nil
}

// ListRooms returns all rooms for user
func (s *Service) ListRooms(ctx context.Context, userID uuid.UUID) ([]*RoomWithUnread, error) {
	rooms, err := s.repo.ListRoomsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*RoomWithUnread, len(rooms))
	for i, room := range rooms {
		unread, _ := s.repo.CountUnreadByRoom(ctx, room.ID, userID)
		result[i] = &RoomWithUnread{
			Room:        room,
			UnreadCount: unread,
		}
	}

	return result, nil
}

// RoomWithUnread room with unread count
type RoomWithUnread struct {
	*Room
	UnreadCount int
}

// SendMessage sends a message in a room
func (s *Service) SendMessage(ctx context.Context, userID, roomID uuid.UUID, req *SendMessageRequest) (*Message, error) {
	if s.limitChecker != nil {
		if err := s.limitChecker.CanUseChat(ctx, userID); err != nil {
			return nil, err
		}
	}

	// Verify room access
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil {
		return nil, ErrRoomNotFound
	}

	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotRoomMember
	}

	// Check if either user has blocked the other (only for direct chats)
	if room.RoomType == RoomTypeDirect {
		// Find other participant
		members, err := s.repo.GetMembers(ctx, room.ID)
		if err != nil {
			return nil, err
		}

		var otherUserID uuid.UUID
		for _, m := range members {
			if m.UserID != userID {
				otherUserID = m.UserID
				break
			}
		}

		if otherUserID != uuid.Nil {
			if err := s.accessChecker.CanCommunicate(ctx, userID, otherUserID); err != nil {
				return nil, err
			}
		}
	}

	msgType := MessageTypeText
	if req.MessageType == "image" {
		msgType = MessageTypeImage
		// Validate image URL - must start with http
		if len(req.Content) < 7 || (req.Content[:7] != "http://" && req.Content[:8] != "https://") {
			return nil, ErrInvalidImageURL
		}
	}

	var attachments []*AttachmentInfo
	if len(req.AttachmentUploadIDs) > 0 {
		for _, uploadID := range req.AttachmentUploadIDs {
			attInfo, err := s.uploadResolver.GetAttachmentInfo(ctx, uploadID, userID)
			if err != nil {
				return nil, err
			}
			attachments = append(attachments, attInfo)

			if req.Content == "" {
				req.Content = attInfo.URL
			}
			// Auto-detect image type from mime if needed
			if attInfo.MimeType != "" && len(attInfo.MimeType) >= 6 && attInfo.MimeType[:6] == "image/" {
				msgType = MessageTypeImage
			}
		}
	}

	msg := &Message{
		ID:          uuid.New(),
		RoomID:      roomID,
		SenderID:    userID,
		Content:     req.Content,
		MessageType: msgType,
		IsRead:      false,
		CreatedAt:   time.Now(),
	}

	if len(attachments) > 0 {
		msg.Attachments = attachments
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Update room's last message
	_ = s.repo.UpdateRoomLastMessage(ctx, roomID, req.Content)

	// Broadcast to WebSocket clients
	if s.hub != nil {
		members, membersErr := s.repo.GetMembers(ctx, roomID)
		if membersErr != nil || len(members) == 0 {
			// Fallback for degraded mode (membership query failed)
			s.hub.BroadcastToRoom(roomID, &WSEvent{Type: EventNewMessage, RoomID: roomID, SenderID: userID, MessageID: msg.ID, Message: msg})
			s.hub.BroadcastToRoom(roomID, &WSEvent{Type: EventMessageCreate, RoomID: roomID, SenderID: userID, MessageID: msg.ID, Message: msg})
			s.hub.BroadcastToRoom(roomID, &WSEvent{
				Type:      EventRoomUpdated,
				RoomID:    roomID,
				SenderID:  userID,
				MessageID: msg.ID,
				Data: map[string]any{
					"last_message_preview": req.Content,
					"last_message_at":      msg.CreatedAt,
				},
			})
		} else {
			for _, member := range members {
				recipientID := member.UserID

				_ = s.hub.SendToUserJSON(recipientID, &WSEvent{
					Type:      EventNewMessage,
					RoomID:    roomID,
					SenderID:  userID,
					MessageID: msg.ID,
					Message:   msg,
				})
				_ = s.hub.SendToUserJSON(recipientID, &WSEvent{
					Type:      EventMessageCreate,
					RoomID:    roomID,
					SenderID:  userID,
					MessageID: msg.ID,
					Message:   msg,
				})

				roomData := map[string]any{
					"last_message_preview": req.Content,
					"last_message_at":      msg.CreatedAt,
				}
				if unreadCount, err := s.repo.CountUnreadByRoom(ctx, roomID, recipientID); err == nil {
					roomData["unread_count"] = unreadCount
				}

				_ = s.hub.SendToUserJSON(recipientID, &WSEvent{
					Type:      EventRoomUpdated,
					RoomID:    roomID,
					SenderID:  userID,
					MessageID: msg.ID,
					Data:      roomData,
				})
			}
		}
	}

	if s.notifService != nil {
		members, err := s.repo.GetMembers(ctx, roomID)
		if err == nil {
			senderName := "Пользователь"
			if sender, err := s.userRepo.GetByID(ctx, userID); err == nil && sender != nil {
				senderName = sender.Email
			}

			preview := req.Content
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			if req.MessageType == "image" {
				preview = "📷 Фото"
			}
			if len(req.AttachmentUploadIDs) > 0 {
				preview = "📎 Вложение"
			}

			for _, member := range members {
				if member.UserID == userID {
					continue
				}
				recipientID := member.UserID
				go func(rid uuid.UUID) {
					_ = s.notifService.NotifyNewMessage(context.Background(), rid, senderName, preview, roomID, msg.ID)
				}(recipientID)
			}
		}
	}

	return msg, nil
}

// GetMessages returns messages for a room with pagination
func (s *Service) GetMessages(ctx context.Context, userID, roomID uuid.UUID, limit, offset int) ([]*Message, error) {
	// Verify room access
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil {
		return nil, ErrRoomNotFound
	}

	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotRoomMember
	}

	return s.repo.ListMessagesByRoom(ctx, roomID, limit, offset)
}

// MarkAsRead marks all messages in room as read
func (s *Service) MarkAsRead(ctx context.Context, userID, roomID uuid.UUID) error {
	// Verify room access
	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil || !isMember {
		return ErrNotRoomMember
	}

	if err := s.repo.MarkMessagesAsRead(ctx, roomID, userID); err != nil {
		return err
	}

	// Broadcast read event via WebSocket
	if s.hub != nil {
		s.hub.BroadcastToRoom(roomID, &WSEvent{
			Type:     EventRead,
			RoomID:   roomID,
			SenderID: userID,
			Data: map[string]any{
				"room_id":   roomID,
				"sender_id": userID,
			},
		})
	}

	return nil
}

// GetUnreadCount returns total unread count for user
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnreadByUser(ctx, userID)
}

// DeleteRoom hard deletes a room and its messages. Only admins or members can do this.
func (s *Service) DeleteRoom(ctx context.Context, userID, roomID uuid.UUID) error {
	// Verify room and access
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil {
		return ErrRoomNotFound
	}
	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrNotRoomMember
	}

	// For group chats, only admin/creator might delete it. For direct, any member can hide/delete.
	// For simplicity, we allow any member of a direct/casting chat to delete the whole room
	// Note: in a pure implementation, you'd hide it for one user, but hard-delete is requested.
	if room.RoomType == RoomTypeGroup {
		member, err := s.repo.GetMember(ctx, roomID, userID)
		if err != nil || member == nil || !member.IsAdmin() {
			return errors.New("only room admins can delete group chats") // Use explicit error
		}
	}

	return s.repo.DeleteRoom(ctx, roomID)
}

// DeleteMessage soft-deletes a message. Only the original sender can do this.
func (s *Service) DeleteMessage(ctx context.Context, userID, messageID uuid.UUID) error {
	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil || msg == nil {
		return errors.New("message not found")
	}

	if msg.SenderID != userID {
		return errors.New("you can only delete your own messages")
	}

	// Soft delete in DB
	err = s.repo.DeleteMessage(ctx, messageID)
	if err != nil {
		return err
	}

	// Broadcast message deletion if Hub is active
	if s.hub != nil {
		s.hub.BroadcastToRoom(msg.RoomID, &WSEvent{
			Type:      EventDeleteMessage,
			RoomID:    msg.RoomID,
			MessageID: msg.ID,
		})
	}

	return nil
}
