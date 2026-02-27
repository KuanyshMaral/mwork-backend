package chat

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

type realtimeRepo struct {
	room    *Room
	members []*RoomMember
}

func (r *realtimeRepo) CreateRoom(context.Context, *Room) error { return nil }
func (r *realtimeRepo) GetRoomByID(ctx context.Context, id uuid.UUID) (*Room, error) {
	if r.room != nil && r.room.ID == id {
		return r.room, nil
	}
	return nil, nil
}
func (r *realtimeRepo) GetDirectRoomByUsers(context.Context, uuid.UUID, uuid.UUID) (*Room, error) {
	return nil, nil
}
func (r *realtimeRepo) ListRoomsByUser(context.Context, uuid.UUID) ([]*Room, error)    { return nil, nil }
func (r *realtimeRepo) UpdateRoomLastMessage(context.Context, uuid.UUID, string) error { return nil }
func (r *realtimeRepo) DeleteRoom(context.Context, uuid.UUID) error                    { return nil }
func (r *realtimeRepo) AddMember(context.Context, *RoomMember) error                   { return nil }
func (r *realtimeRepo) RemoveMember(context.Context, uuid.UUID, uuid.UUID) error       { return nil }
func (r *realtimeRepo) GetMembers(context.Context, uuid.UUID) ([]*RoomMember, error) {
	return r.members, nil
}
func (r *realtimeRepo) GetMember(context.Context, uuid.UUID, uuid.UUID) (*RoomMember, error) {
	return nil, nil
}
func (r *realtimeRepo) IsMember(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (r *realtimeRepo) UpdateMemberRole(context.Context, uuid.UUID, uuid.UUID, MemberRole) error {
	return nil
}
func (r *realtimeRepo) HasCastingResponseAccess(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (r *realtimeRepo) CreateMessage(context.Context, *Message) error               { return nil }
func (r *realtimeRepo) GetMessageByID(context.Context, uuid.UUID) (*Message, error) { return nil, nil }
func (r *realtimeRepo) ListMessagesByRoom(context.Context, uuid.UUID, int, int) ([]*Message, error) {
	return nil, nil
}
func (r *realtimeRepo) DeleteMessage(context.Context, uuid.UUID) error                 { return nil }
func (r *realtimeRepo) MarkMessagesAsRead(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (r *realtimeRepo) CountUnreadByRoom(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	return 0, nil
}
func (r *realtimeRepo) CountUnreadByUser(context.Context, uuid.UUID) (int, error) { return 0, nil }

type noopAccessChecker struct{}

func (n *noopAccessChecker) CanCommunicate(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type staticUploadResolver struct{}

func (s *staticUploadResolver) GetAttachmentInfo(context.Context, uuid.UUID, uuid.UUID) (*AttachmentInfo, error) {
	return &AttachmentInfo{URL: "https://cdn.example.com/file.png", FileName: "file.png", MimeType: "image/png", Size: 12}, nil
}

func waitEvent(t *testing.T, ch <-chan []byte) WSEvent {
	t.Helper()
	select {
	case msg := <-ch:
		var event WSEvent
		if err := json.Unmarshal(msg, &event); err != nil {
			t.Fatalf("unmarshal ws event: %v", err)
		}
		return event
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting websocket event")
	}
	return WSEvent{}
}

func newLocalHubWithUsers(roomID uuid.UUID, userIDs ...uuid.UUID) (*Hub, map[uuid.UUID]*Connection) {
	h := NewHub(nil)
	conns := map[uuid.UUID]*Connection{}
	h.connections = make(map[uuid.UUID]map[*Connection]bool)
	h.localRooms = map[uuid.UUID]map[uuid.UUID]bool{roomID: {}}
	for _, userID := range userIDs {
		conn := &Connection{UserID: userID, Send: make(chan []byte, 4)}
		conns[userID] = conn
		h.connections[userID] = map[*Connection]bool{conn: true}
		h.localRooms[roomID][userID] = true
	}
	return h, conns
}

func TestSendMessageBroadcastsRealtimeWithAttachments(t *testing.T) {
	sender := uuid.New()
	recipient := uuid.New()
	roomID := uuid.New()
	hub, conns := newLocalHubWithUsers(roomID, sender, recipient)

	repo := &realtimeRepo{
		room: &Room{ID: roomID, RoomType: RoomTypeDirect},
		members: []*RoomMember{
			{RoomID: roomID, UserID: sender, Role: MemberRoleMember},
			{RoomID: roomID, UserID: recipient, Role: MemberRoleMember},
		},
	}
	users := &testUserRepo{users: map[uuid.UUID]*user.User{
		sender:    {ID: sender, Email: "sender@example.com"},
		recipient: {ID: recipient, Email: "recipient@example.com"},
	}}
	svc := NewService(repo, users, hub, &noopAccessChecker{}, nil, &staticUploadResolver{})

	_, err := svc.SendMessage(context.Background(), sender, roomID, &SendMessageRequest{AttachmentUploadIDs: []uuid.UUID{uuid.New()}})
	if err != nil {
		t.Fatalf("send message: %v", err)
	}

	event := waitEvent(t, conns[recipient].Send)
	if event.Type != EventNewMessage {
		t.Fatalf("expected %s, got %s", EventNewMessage, event.Type)
	}
	if event.Message == nil || len(event.Message.Attachments) != 1 {
		t.Fatalf("expected attachment in event payload")
	}

	eventCreated := waitEvent(t, conns[recipient].Send)
	if eventCreated.Type != EventMessageCreate {
		t.Fatalf("expected %s, got %s", EventMessageCreate, eventCreated.Type)
	}
	if eventCreated.Message == nil || len(eventCreated.Message.Attachments) != 1 {
		t.Fatalf("expected attachment in message_created payload")
	}

	eventRoom := waitEvent(t, conns[recipient].Send)
	if eventRoom.Type != EventRoomUpdated {
		t.Fatalf("expected %s, got %s", EventRoomUpdated, eventRoom.Type)
	}
	if eventRoom.RoomID != roomID {
		t.Fatalf("expected room_id %s, got %s", roomID, eventRoom.RoomID)
	}
	data, ok := eventRoom.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected room_updated data map, got %T", eventRoom.Data)
	}
	if _, ok := data["last_message_preview"]; !ok {
		t.Fatalf("expected last_message_preview in room_updated data")
	}
	if _, ok := data["last_message_at"]; !ok {
		t.Fatalf("expected last_message_at in room_updated data")
	}
}

func TestMarkAsReadBroadcastsRealtime(t *testing.T) {
	reader := uuid.New()
	other := uuid.New()
	roomID := uuid.New()
	hub, conns := newLocalHubWithUsers(roomID, reader, other)

	repo := &realtimeRepo{room: &Room{ID: roomID, RoomType: RoomTypeDirect}}
	svc := NewService(repo, &testUserRepo{}, hub, &noopAccessChecker{}, nil, &staticUploadResolver{})

	if err := svc.MarkAsRead(context.Background(), reader, roomID); err != nil {
		t.Fatalf("mark as read: %v", err)
	}

	event := waitEvent(t, conns[other].Send)
	if event.Type != EventRead {
		t.Fatalf("expected %s, got %s", EventRead, event.Type)
	}
	if event.SenderID != reader {
		t.Fatalf("expected sender %s, got %s", reader, event.SenderID)
	}
}
