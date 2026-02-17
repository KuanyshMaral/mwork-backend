package chat

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines chat data access interface
type Repository interface {
	// Room operations
	CreateRoom(ctx context.Context, room *Room) error
	GetRoomByID(ctx context.Context, id uuid.UUID) (*Room, error)
	GetRoomByParticipants(ctx context.Context, user1, user2 uuid.UUID) (*Room, error)
	ListRoomsByUser(ctx context.Context, userID uuid.UUID) ([]*Room, error)
	UpdateRoomLastMessage(ctx context.Context, roomID uuid.UUID, preview string) error
	DeleteRoom(ctx context.Context, id uuid.UUID) error
	HasCastingResponseAccess(ctx context.Context, castingID, user1, user2 uuid.UUID) (bool, error)

	// Message operations
	CreateMessage(ctx context.Context, msg *Message) error
	GetMessageByID(ctx context.Context, id uuid.UUID) (*Message, error)
	ListMessagesByRoom(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*Message, error)
	DeleteMessage(ctx context.Context, id uuid.UUID) error
	MarkMessagesAsRead(ctx context.Context, roomID, userID uuid.UUID) error
	CountUnreadByRoom(ctx context.Context, roomID, userID uuid.UUID) (int, error)
	CountUnreadByUser(ctx context.Context, userID uuid.UUID) (int, error)
}

type repository struct {
	db *sqlx.DB
}

// NewRepository creates new chat repository
func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

// Room operations

func (r *repository) CreateRoom(ctx context.Context, room *Room) error {
	query := `
		INSERT INTO chat_rooms (id, participant1_id, participant2_id, casting_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		room.ID,
		room.Participant1ID,
		room.Participant2ID,
		room.CastingID,
		room.CreatedAt,
	)
	return err
}

func (r *repository) GetRoomByID(ctx context.Context, id uuid.UUID) (*Room, error) {
	query := `SELECT * FROM chat_rooms WHERE id = $1`
	var room Room
	err := r.db.GetContext(ctx, &room, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &room, nil
}

func (r *repository) GetRoomByParticipants(ctx context.Context, user1, user2 uuid.UUID) (*Room, error) {
	// Ensure consistent ordering
	p1, p2 := user1, user2
	if p1.String() > p2.String() {
		p1, p2 = p2, p1
	}

	query := `SELECT * FROM chat_rooms WHERE participant1_id = $1 AND participant2_id = $2`
	var room Room
	err := r.db.GetContext(ctx, &room, query, p1, p2)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &room, nil
}

func (r *repository) ListRoomsByUser(ctx context.Context, userID uuid.UUID) ([]*Room, error) {
	query := `
		SELECT * FROM chat_rooms 
		WHERE participant1_id = $1 OR participant2_id = $1
		ORDER BY last_message_at DESC NULLS LAST, created_at DESC
	`
	var rooms []*Room
	err := r.db.SelectContext(ctx, &rooms, query, userID)
	return rooms, err
}

func (r *repository) UpdateRoomLastMessage(ctx context.Context, roomID uuid.UUID, preview string) error {
	// Truncate preview to 100 chars
	if len(preview) > 97 {
		preview = preview[:97] + "..."
	}

	query := `
		UPDATE chat_rooms 
		SET last_message_at = NOW(), last_message_preview = $2
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, roomID, preview)
	return err
}

// Message operations

func (r *repository) CreateMessage(ctx context.Context, msg *Message) error {
	query := `
		INSERT INTO messages (id, room_id, sender_id, content, message_type, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		msg.ID,
		msg.RoomID,
		msg.SenderID,
		msg.Content,
		msg.MessageType,
		msg.IsRead,
		msg.CreatedAt,
	)
	return err
}

func (r *repository) GetMessageByID(ctx context.Context, id uuid.UUID) (*Message, error) {
	query := `SELECT * FROM messages WHERE id = $1 AND deleted_at IS NULL`
	var msg Message
	err := r.db.GetContext(ctx, &msg, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}

func (r *repository) ListMessagesByRoom(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*Message, error) {
	query := `
		SELECT * FROM messages 
		WHERE room_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	var messages []*Message
	err := r.db.SelectContext(ctx, &messages, query, roomID, limit, offset)
	return messages, err
}

func (r *repository) MarkMessagesAsRead(ctx context.Context, roomID, userID uuid.UUID) error {
	query := `
		UPDATE messages 
		SET is_read = true, read_at = NOW()
		WHERE room_id = $1 
		  AND sender_id != $2 
		  AND NOT is_read 
		  AND deleted_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, roomID, userID)
	return err
}

func (r *repository) CountUnreadByRoom(ctx context.Context, roomID, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM messages 
		WHERE room_id = $1 AND sender_id != $2 AND NOT is_read AND deleted_at IS NULL
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, roomID, userID)
	return count, err
}

func (r *repository) CountUnreadByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*) FROM messages m
		JOIN chat_rooms r ON m.room_id = r.id
		WHERE (r.participant1_id = $1 OR r.participant2_id = $1)
		AND m.sender_id != $1 AND NOT m.is_read AND m.deleted_at IS NULL
	`
	var count int
	err := r.db.GetContext(ctx, &count, query, userID)
	return count, err
}

// DeleteMessage soft deletes a message
func (r *repository) DeleteMessage(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE messages SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DeleteRoom deletes a chat room and its messages
func (r *repository) DeleteRoom(ctx context.Context, id uuid.UUID) error {
	// Delete all messages in the room first
	_, err := r.db.ExecContext(ctx, `DELETE FROM messages WHERE room_id = $1`, id)
	if err != nil {
		return err
	}
	// Delete the room
	_, err = r.db.ExecContext(ctx, `DELETE FROM chat_rooms WHERE id = $1`, id)
	return err
}

// HasCastingResponseAccess checks if users can access chat for a casting through a response-owner pair.
func (r *repository) HasCastingResponseAccess(ctx context.Context, castingID, user1, user2 uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM castings c
			JOIN casting_responses cr ON cr.casting_id = c.id
			WHERE c.id = $1
			  AND (
				(cr.user_id = $2 AND c.creator_id = $3)
				OR
				(cr.user_id = $3 AND c.creator_id = $2)
			  )
		)
	`

	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, castingID, user1, user2); err != nil {
		return false, err
	}

	return exists, nil
}
