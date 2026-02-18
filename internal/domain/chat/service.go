package chat

import (
	"context"
	"database/sql"
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
	IsCommitted(ctx context.Context, uploadID uuid.UUID) (bool, error)
	GetUploadURL(ctx context.Context, uploadID uuid.UUID) (string, error)
	CommitUpload(ctx context.Context, uploadID, userID uuid.UUID) (*AttachmentInfo, error)
}

// Service handles chat business logic
type Service struct {
	repo           Repository
	userRepo       user.Repository
	hub            *Hub // WebSocket hub
	accessChecker  AccessChecker
	limitChecker   LimitChecker
	uploadResolver UploadResolver
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
		if req.RecipientID != nil {
			return s.CreateCastingRoom(ctx, userID, *req.CastingID, []uuid.UUID{*req.RecipientID}, req.Name)
		}
		return s.CreateCastingRoom(ctx, userID, *req.CastingID, req.MemberIDs, req.Name)

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

	var attachmentInfo *AttachmentInfo
	if req.AttachmentUploadID != nil {
		var err error
		attachmentInfo, err = s.uploadResolver.CommitUpload(ctx, *req.AttachmentUploadID, userID)
		if err != nil {
			return nil, err
		}

		if req.Content == "" {
			req.Content = attachmentInfo.URL
		}
		// Auto-detect image type from mime if needed, or stick to what client sent
		if attachmentInfo.MimeType != "" && len(attachmentInfo.MimeType) >= 6 && attachmentInfo.MimeType[:6] == "image/" {
			msgType = MessageTypeImage
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

	if attachmentInfo != nil {
		msg.AttachmentUploadID = uuid.NullUUID{UUID: attachmentInfo.UploadID, Valid: true}
		msg.AttachmentURL = sql.NullString{String: attachmentInfo.URL, Valid: true}
		msg.AttachmentName = sql.NullString{String: attachmentInfo.FileName, Valid: true}
		msg.AttachmentMime = sql.NullString{String: attachmentInfo.MimeType, Valid: true}
		msg.AttachmentSize = sql.NullInt64{Int64: attachmentInfo.Size, Valid: true}
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Update room's last message
	_ = s.repo.UpdateRoomLastMessage(ctx, roomID, req.Content)

	// Broadcast to WebSocket clients
	if s.hub != nil {
		s.hub.BroadcastToRoom(roomID, &WSEvent{
			Type:    EventNewMessage,
			RoomID:  roomID,
			Message: msg,
		})
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
		})
	}

	return nil
}

// GetUnreadCount returns total unread count for user
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnreadByUser(ctx, userID)
}
