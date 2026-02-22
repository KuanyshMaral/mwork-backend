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
	GetDirectRoomByUsers(ctx context.Context, user1, user2 uuid.UUID) (*Room, error)
	ListRoomsByUser(ctx context.Context, userID uuid.UUID) ([]*Room, error)
	UpdateRoomLastMessage(ctx context.Context, roomID uuid.UUID, preview string) error
	DeleteRoom(ctx context.Context, id uuid.UUID) error

	// Member operations
	AddMember(ctx context.Context, member *RoomMember) error
	RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error
	GetMembers(ctx context.Context, roomID uuid.UUID) ([]*RoomMember, error)
	GetMember(ctx context.Context, roomID, userID uuid.UUID) (*RoomMember, error)
	IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	UpdateMemberRole(ctx context.Context, roomID, userID uuid.UUID, role MemberRole) error

	// Casting access
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
		INSERT INTO chat_rooms (id, room_type, name, creator_id, casting_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		room.ID,
		room.RoomType,
		room.Name,
		room.CreatorID,
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

func (r *repository) GetDirectRoomByUsers(ctx context.Context, user1, user2 uuid.UUID) (*Room, error) {
	query := `
		SELECT r.* FROM chat_rooms r
		WHERE r.room_type = 'direct'
		AND EXISTS (SELECT 1 FROM chat_room_members WHERE room_id = r.id AND user_id = $1)
		AND EXISTS (SELECT 1 FROM chat_room_members WHERE room_id = r.id AND user_id = $2)
	`
	var room Room
	err := r.db.GetContext(ctx, &room, query, user1, user2)
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
		SELECT DISTINCT r.* FROM chat_rooms r
		JOIN chat_room_members m ON m.room_id = r.id
		WHERE m.user_id = $1
		ORDER BY r.last_message_at DESC NULLS LAST, r.created_at DESC
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
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO messages (id, room_id, sender_id, content, message_type, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.ExecContext(ctx, query,
		msg.ID,
		msg.RoomID,
		msg.SenderID,
		msg.Content,
		msg.MessageType,
		msg.IsRead,
		msg.CreatedAt,
	)
	if err != nil {
		return err
	}

	// Insert into polymorphic attachments table
	if len(msg.Attachments) > 0 {
		attachmentQuery := `
			INSERT INTO attachments (upload_id, target_id, target_type, sort_order)
			VALUES ($1, $2, $3, $4)
		`
		for i, att := range msg.Attachments {
			_, err = tx.ExecContext(ctx, attachmentQuery,
				att.UploadID, msg.ID, "chat_attachment", i)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *repository) GetMessageByID(ctx context.Context, id uuid.UUID) (*Message, error) {
	query := `
		SELECT m.*
		FROM messages m
		WHERE m.id = $1 AND m.deleted_at IS NULL
	`
	var msg Message
	err := r.db.GetContext(ctx, &msg, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Load attachments
	err = r.loadAttachments(ctx, []*Message{&msg})
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

func (r *repository) ListMessagesByRoom(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*Message, error) {
	query := `
		SELECT m.*
		FROM messages m
		WHERE m.room_id = $1 AND m.deleted_at IS NULL
		ORDER BY m.created_at DESC
		LIMIT $2 OFFSET $3
	`
	var messages []*Message
	err := r.db.SelectContext(ctx, &messages, query, roomID, limit, offset)
	if err != nil {
		return nil, err
	}

	if len(messages) > 0 {
		if err := r.loadAttachments(ctx, messages); err != nil {
			return nil, err
		}
	}

	return messages, nil
}

type attachmentRow struct {
	MessageID uuid.UUID `db:"target_id"`
	UploadID  uuid.UUID `db:"upload_id"`
	URL       string    `db:"attachment_url"`
	FileName  string    `db:"attachment_name"`
	MimeType  string    `db:"attachment_mime"`
	Size      int64     `db:"attachment_size"`
}

func (r *repository) loadAttachments(ctx context.Context, messages []*Message) error {
	if len(messages) == 0 {
		return nil
	}

	msgIDs := make([]uuid.UUID, len(messages))
	msgMap := make(map[uuid.UUID]*Message)
	for i, m := range messages {
		msgIDs[i] = m.ID
		msgMap[m.ID] = m
		m.Attachments = make([]*AttachmentInfo, 0)
	}

	query, args, err := sqlx.In(`
		SELECT a.target_id, a.upload_id, 
			   u.file_path as attachment_url, u.original_name as attachment_name, 
			   u.mime_type as attachment_mime, u.size_bytes as attachment_size
		FROM attachments a
		JOIN uploads u ON a.upload_id = u.id
		WHERE a.target_type = 'chat_attachment' AND a.target_id IN (?)
		ORDER BY a.sort_order ASC
	`, msgIDs)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)

	var rows []attachmentRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return err
	}

	for _, row := range rows {
		if msg, ok := msgMap[row.MessageID]; ok {
			msg.Attachments = append(msg.Attachments, &AttachmentInfo{
				UploadID: row.UploadID,
				URL:      row.URL,
				FileName: row.FileName,
				MimeType: row.MimeType,
				Size:     row.Size,
			})
		}
	}

	return nil
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
		JOIN chat_room_members crm ON m.room_id = crm.room_id
		WHERE crm.user_id = $1
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
