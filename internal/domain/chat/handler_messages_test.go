package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
)

type getMessagesRepo struct {
	room     *Room
	messages []*Message
}

func (r *getMessagesRepo) CreateRoom(context.Context, *Room) error { return nil }
func (r *getMessagesRepo) GetRoomByID(context.Context, uuid.UUID) (*Room, error) {
	return r.room, nil
}
func (r *getMessagesRepo) GetDirectRoomByUsers(context.Context, uuid.UUID, uuid.UUID) (*Room, error) {
	return nil, nil
}
func (r *getMessagesRepo) ListRoomsByUser(context.Context, uuid.UUID) ([]*Room, error) {
	return nil, nil
}
func (r *getMessagesRepo) UpdateRoomLastMessage(context.Context, uuid.UUID, string) error {
	return nil
}
func (r *getMessagesRepo) DeleteRoom(context.Context, uuid.UUID) error              { return nil }
func (r *getMessagesRepo) AddMember(context.Context, *RoomMember) error             { return nil }
func (r *getMessagesRepo) RemoveMember(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *getMessagesRepo) GetMembers(context.Context, uuid.UUID) ([]*RoomMember, error) {
	return nil, nil
}
func (r *getMessagesRepo) GetMember(context.Context, uuid.UUID, uuid.UUID) (*RoomMember, error) {
	return nil, nil
}
func (r *getMessagesRepo) IsMember(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (r *getMessagesRepo) UpdateMemberRole(context.Context, uuid.UUID, uuid.UUID, MemberRole) error {
	return nil
}
func (r *getMessagesRepo) HasCastingResponseAccess(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (r *getMessagesRepo) CreateMessage(context.Context, *Message) error { return nil }
func (r *getMessagesRepo) GetMessageByID(context.Context, uuid.UUID) (*Message, error) {
	return nil, nil
}
func (r *getMessagesRepo) ListMessagesByRoom(context.Context, uuid.UUID, int, int) ([]*Message, error) {
	return r.messages, nil
}
func (r *getMessagesRepo) DeleteMessage(context.Context, uuid.UUID) error                 { return nil }
func (r *getMessagesRepo) MarkMessagesAsRead(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *getMessagesRepo) CountUnreadByRoom(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	return 0, nil
}
func (r *getMessagesRepo) CountUnreadByUser(context.Context, uuid.UUID) (int, error) { return 0, nil }

func TestHandlerGetMessages_Returns200WithLegacyAttachmentUploadID(t *testing.T) {
	userID := uuid.New()
	roomID := uuid.New()
	uploadID := uuid.New()

	repo := &getMessagesRepo{
		room: &Room{ID: roomID, RoomType: RoomTypeDirect, CreatedAt: time.Now()},
		messages: []*Message{{
			ID:                 uuid.New(),
			RoomID:             roomID,
			SenderID:           userID,
			Content:            "legacy attachment",
			MessageType:        MessageTypeText,
			CreatedAt:          time.Now(),
			AttachmentUploadID: &uploadID,
		}},
	}

	svc := NewService(repo, nil, nil, nil, nil, nil)
	h := NewHandler(svc, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chat/rooms/"+roomID.String()+"/messages?limit=50&offset=0", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", roomID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID)

	rr := httptest.NewRecorder()
	h.GetMessages(rr, req.WithContext(ctx))

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload struct {
		Success bool              `json:"success"`
		Data    []MessageResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !payload.Success {
		t.Fatal("expected success=true")
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected 1 message, got %d", len(payload.Data))
	}
}
