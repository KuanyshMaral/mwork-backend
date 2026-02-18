package chat

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

// CreateDirectRoom creates a 1-to-1 direct chat room
func (s *Service) CreateDirectRoom(ctx context.Context, userID uuid.UUID, recipientID uuid.UUID, castingID *uuid.UUID) (*Room, error) {
	// Validation
	if userID == recipientID {
		return nil, ErrCannotChatSelf
	}

	// Check sender exists and is verified (if employer/agency)
	sender, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || sender == nil {
		return nil, ErrUserNotFound
	}
	if (sender.Role == user.RoleEmployer || sender.Role == user.RoleAgency) && !sender.IsVerificationApproved() {
		return nil, ErrEmployerNotVerified
	}

	// Check recipient exists
	recipient, err := s.userRepo.GetByID(ctx, recipientID)
	if err != nil || recipient == nil {
		return nil, ErrUserNotFound
	}

	// Check access (blocks and bans)
	if s.accessChecker != nil {
		if err := s.accessChecker.CanCommunicate(ctx, userID, recipientID); err != nil {
			return nil, err
		}
	}

	// Check casting access OR subscription limits
	if castingID != nil {
		// Casting chat - check HasCastingResponseAccess
		hasAccess, err := s.repo.HasCastingResponseAccess(ctx, *castingID, userID, recipientID)
		if err != nil {
			return nil, err
		}
		if !hasAccess {
			return nil, ErrNoAccess
		}
	} else {
		// Regular direct chat - check subscription limits
		if s.limitChecker != nil {
			if err := s.limitChecker.CanUseChat(ctx, userID); err != nil {
				return nil, err
			}
		}
	}

	// Check if direct room already exists between these users
	existing, err := s.repo.GetDirectRoomByUsers(ctx, userID, recipientID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// Create new direct room
	room := &Room{
		ID:        uuid.New(),
		RoomType:  RoomTypeDirect,
		CreatedAt: time.Now(),
	}

	if castingID != nil {
		room.CastingID = uuid.NullUUID{UUID: *castingID, Valid: true}
	}

	if err := s.repo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}

	// Add both members
	if err := s.repo.AddMember(ctx, &RoomMember{
		ID:       uuid.New(),
		RoomID:   room.ID,
		UserID:   userID,
		Role:     MemberRoleMember,
		JoinedAt: time.Now(),
	}); err != nil {
		return nil, err
	}

	if err := s.repo.AddMember(ctx, &RoomMember{
		ID:       uuid.New(),
		RoomID:   room.ID,
		UserID:   recipientID,
		Role:     MemberRoleMember,
		JoinedAt: time.Now(),
	}); err != nil {
		return nil, err
	}

	return room, nil
}

// CreateCastingRoom creates a casting-related chat room (employer can add multiple models)
func (s *Service) CreateCastingRoom(ctx context.Context, userID uuid.UUID, castingID uuid.UUID, memberIDs []uuid.UUID, name string) (*Room, error) {
	// Verify user has access to create casting room (should be casting owner)
	// This will be checked by HasCastingResponseAccess in iteration

	if len(memberIDs) == 0 {
		return nil, ErrRoomNotFound // need at least 1 other member
	}

	// Create casting room
	room := &Room{
		ID:        uuid.New(),
		RoomType:  RoomTypeCasting,
		Name:      sql.NullString{String: name, Valid: name != ""},
		CreatorID: uuid.NullUUID{UUID: userID, Valid: true},
		CastingID: uuid.NullUUID{UUID: castingID, Valid: true},
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}

	// Add creator as admin
	if err := s.repo.AddMember(ctx, &RoomMember{
		ID:       uuid.New(),
		RoomID:   room.ID,
		UserID:   userID,
		Role:     MemberRoleAdmin,
		JoinedAt: time.Now(),
	}); err != nil {
		return nil, err
	}

	// Add other members
	for _, memberID := range memberIDs {
		if memberID == userID {
			continue // skip creator
		}
		if err := s.repo.AddMember(ctx, &RoomMember{
			ID:       uuid.New(),
			RoomID:   room.ID,
			UserID:   memberID,
			Role:     MemberRoleMember,
			JoinedAt: time.Now(),
		}); err != nil {
			return nil, err
		}
	}

	return room, nil
}

// CreateGroupRoom creates a multi-user group chat
func (s *Service) CreateGroupRoom(ctx context.Context, userID uuid.UUID, memberIDs []uuid.UUID, name string) (*Room, error) {
	if len(memberIDs) == 0 {
		return nil, ErrRoomNotFound
	}

	// Check creator has chat access
	if s.limitChecker != nil {
		if err := s.limitChecker.CanUseChat(ctx, userID); err != nil {
			return nil, err
		}
	}

	// Create group room
	room := &Room{
		ID:        uuid.New(),
		RoomType:  RoomTypeGroup,
		Name:      sql.NullString{String: name, Valid: true},
		CreatorID: uuid.NullUUID{UUID: userID, Valid: true},
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}

	// Add creator as admin
	if err := s.repo.AddMember(ctx, &RoomMember{
		ID:       uuid.New(),
		RoomID:   room.ID,
		UserID:   userID,
		Role:     MemberRoleAdmin,
		JoinedAt: time.Now(),
	}); err != nil {
		return nil, err
	}

	// Add other members
	for _, memberID := range memberIDs {
		if memberID == userID {
			continue
		}
		if err := s.repo.AddMember(ctx, &RoomMember{
			ID:       uuid.New(),
			RoomID:   room.ID,
			UserID:   memberID,
			Role:     MemberRoleMember,
			JoinedAt: time.Now(),
		}); err != nil {
			return nil, err
		}
	}

	return room, nil
}

// AddMember adds a user to a room (admin only for group/casting rooms)
func (s *Service) AddMember(ctx context.Context, userID, roomID, newMemberID uuid.UUID) error {
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil {
		return ErrRoomNotFound
	}

	// Check requester is member of room
	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil || !isMember {
		return ErrNotRoomMember
	}

	// For group/casting rooms, only admin can add members
	if room.RoomType == RoomTypeGroup || room.RoomType == RoomTypeCasting {
		member, err := s.repo.GetMember(ctx, roomID, userID)
		if err != nil || member == nil || !member.IsAdmin() {
			return ErrNotRoomAdmin
		}
	}

	// Check if already member
	alreadyMember, err := s.repo.IsMember(ctx, roomID, newMemberID)
	if err != nil {
		return err
	}
	if alreadyMember {
		return ErrAlreadyMember
	}

	// Add member
	return s.repo.AddMember(ctx, &RoomMember{
		ID:       uuid.New(),
		RoomID:   roomID,
		UserID:   newMemberID,
		Role:     MemberRoleMember,
		JoinedAt: time.Now(),
	})
}

// RemoveMember removes a user from a room (admin only, or self-leave)
func (s *Service) RemoveMember(ctx context.Context, userID, roomID, targetUserID uuid.UUID) error {
	room, err := s.repo.GetRoomByID(ctx, roomID)
	if err != nil || room == nil {
		return ErrRoomNotFound
	}

	// Check requester is member
	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil || !isMember {
		return ErrNotRoomMember
	}

	// Self-leave is allowed
	if userID == targetUserID {
		return s.repo.RemoveMember(ctx, roomID, targetUserID)
	}

	// Only admin can remove others
	member, err := s.repo.GetMember(ctx, roomID, userID)
	if err != nil || member == nil || !member.IsAdmin() {
		return ErrNotRoomAdmin
	}

	return s.repo.RemoveMember(ctx, roomID, targetUserID)
}

// GetMembers returns all members of a room
func (s *Service) GetMembers(ctx context.Context, userID, roomID uuid.UUID) ([]*RoomMember, error) {
	// Check requester is member
	isMember, err := s.repo.IsMember(ctx, roomID, userID)
	if err != nil || !isMember {
		return nil, ErrNotRoomMember
	}

	return s.repo.GetMembers(ctx, roomID)
}
