package chat

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

// Member operations

func (r *repository) AddMember(ctx context.Context, member *RoomMember) error {
	query := `
		INSERT INTO chat_room_members (id, room_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		member.ID,
		member.RoomID,
		member.UserID,
		member.Role,
		member.JoinedAt,
	)
	return err
}

func (r *repository) RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error {
	query := `DELETE FROM chat_room_members WHERE room_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, roomID, userID)
	return err
}

func (r *repository) GetMembers(ctx context.Context, roomID uuid.UUID) ([]*RoomMember, error) {
	query := `SELECT * FROM chat_room_members WHERE room_id = $1 ORDER BY joined_at ASC`
	var members []*RoomMember
	err := r.db.SelectContext(ctx, &members, query, roomID)
	return members, err
}

func (r *repository) GetMember(ctx context.Context, roomID, userID uuid.UUID) (*RoomMember, error) {
	query := `SELECT * FROM chat_room_members WHERE room_id = $1 AND user_id = $2`
	var member RoomMember
	err := r.db.GetContext(ctx, &member, query, roomID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &member, nil
}

func (r *repository) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM chat_room_members WHERE room_id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.GetContext(ctx, &exists, query, roomID, userID)
	return exists, err
}

func (r *repository) UpdateMemberRole(ctx context.Context, roomID, userID uuid.UUID, role MemberRole) error {
	query := `UPDATE chat_room_members SET role = $1 WHERE room_id = $2 AND user_id = $3`
	_, err := r.db.ExecContext(ctx, query, role, roomID, userID)
	return err
}
