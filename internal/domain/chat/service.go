package chat

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

// BlockChecker defines interface for checking if users are blocked
type BlockChecker interface {
	IsBlocked(ctx context.Context, user1, user2 uuid.UUID) (bool, error)
}

// LimitChecker defines interface for subscription limit checks.
type LimitChecker interface {
	CanUseChat(ctx context.Context, userID uuid.UUID) error
}

// Service handles chat business logic
type Service struct {
	repo         Repository
	userRepo     user.Repository
	hub          *Hub // WebSocket hub
	blockChecker BlockChecker
	limitChecker LimitChecker
}

// NewService creates chat service
func NewService(repo Repository, userRepo user.Repository, hub *Hub, blockChecker BlockChecker, limitChecker LimitChecker) *Service {
	return &Service{
		repo:         repo,
		userRepo:     userRepo,
		hub:          hub,
		blockChecker: blockChecker,
		limitChecker: limitChecker,
	}
}

// CreateOrGetRoom creates a room or returns existing one
func (s *Service) CreateOrGetRoom(ctx context.Context, userID uuid.UUID, req *CreateRoomRequest) (*Room, error) {
	if s.limitChecker != nil {
		if err := s.limitChecker.CanUseChat(ctx, userID); err != nil {
			return nil, err
		}
	}

	// Can't chat with yourself
	if userID == req.RecipientID {
		return nil, ErrCannotChatSelf
	}

	// Ensure employers/agencies are verified before using chat
	sender, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || sender == nil {
		return nil, ErrUserNotFound
	}
	if (sender.Role == user.RoleEmployer || sender.Role == user.RoleAgency) && !sender.IsVerificationApproved() {
		return nil, ErrEmployerNotVerified
	}

	// Check recipient exists
	recipient, err := s.userRepo.GetByID(ctx, req.RecipientID)
	if err != nil || recipient == nil {
		return nil, ErrUserNotFound
	}

	// Check if either user has blocked the other
	if s.blockChecker != nil {
		blocked, err := s.blockChecker.IsBlocked(ctx, userID, req.RecipientID)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, ErrUserBlocked
		}
	}

	// Check if room already exists
	existing, err := s.repo.GetRoomByParticipants(ctx, userID, req.RecipientID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Create new room (ensure consistent ordering)
	p1, p2 := userID, req.RecipientID
	if p1.String() > p2.String() {
		p1, p2 = p2, p1
	}

	room := &Room{
		ID:             uuid.New(),
		Participant1ID: p1,
		Participant2ID: p2,
		CreatedAt:      time.Now(),
	}

	if req.CastingID != nil {
		room.CastingID = uuid.NullUUID{UUID: *req.CastingID, Valid: true}
	}

	if err := s.repo.CreateRoom(ctx, room); err != nil {
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
}

// GetRoom returns room by ID
func (s *Service) GetRoom(ctx context.Context, userID, roomID uuid.UUID) (*Room, error) {
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil {
		return nil, ErrRoomNotFound
	}

	if !room.HasParticipant(userID) {
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

	if !room.HasParticipant(userID) {
		return nil, ErrNotRoomMember
	}

	// Check if either user has blocked the other
	otherUserID := room.GetOtherParticipant(userID)
	if s.blockChecker != nil {
		blocked, err := s.blockChecker.IsBlocked(ctx, userID, otherUserID)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, ErrUserBlocked
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

	msg := &Message{
		ID:          uuid.New(),
		RoomID:      roomID,
		SenderID:    userID,
		Content:     req.Content,
		MessageType: msgType,
		IsRead:      false,
		CreatedAt:   time.Now(),
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

	if !room.HasParticipant(userID) {
		return nil, ErrNotRoomMember
	}

	return s.repo.ListMessagesByRoom(ctx, roomID, limit, offset)
}

// MarkAsRead marks all messages in room as read
func (s *Service) MarkAsRead(ctx context.Context, userID, roomID uuid.UUID) error {
	// Verify room access
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil {
		return ErrRoomNotFound
	}

	if !room.HasParticipant(userID) {
		return ErrNotRoomMember
	}

	if err := s.repo.MarkMessagesAsRead(ctx, roomID, userID); err != nil {
		return err
	}

	// Broadcast read event to other participant via WebSocket
	if s.hub != nil {
		otherUserID := room.GetOtherParticipant(userID)
		s.hub.BroadcastToRoom(roomID, &WSEvent{
			Type:     EventRead,
			RoomID:   roomID,
			SenderID: otherUserID, // Notify the other user
		})
	}

	return nil
}

// GetUnreadCount returns total unread count for user
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountUnreadByUser(ctx, userID)
}
