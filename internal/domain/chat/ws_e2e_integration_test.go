package chat

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/middleware"
	jwtpkg "github.com/mwork/mwork-api/internal/pkg/jwt"
)

type wsE2ERepo struct {
	mu      sync.RWMutex
	room    *Room
	members []*RoomMember
}

func (r *wsE2ERepo) CreateRoom(context.Context, *Room) error { return nil }
func (r *wsE2ERepo) GetRoomByID(context.Context, uuid.UUID) (*Room, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.room, nil
}
func (r *wsE2ERepo) GetDirectRoomByUsers(context.Context, uuid.UUID, uuid.UUID) (*Room, error) {
	return nil, nil
}
func (r *wsE2ERepo) ListRoomsByUser(ctx context.Context, userID uuid.UUID) ([]*Room, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.members {
		if m.UserID == userID {
			return []*Room{r.room}, nil
		}
	}
	return nil, nil
}
func (r *wsE2ERepo) UpdateRoomLastMessage(context.Context, uuid.UUID, string) error { return nil }
func (r *wsE2ERepo) DeleteRoom(context.Context, uuid.UUID) error                    { return nil }
func (r *wsE2ERepo) AddMember(context.Context, *RoomMember) error                   { return nil }
func (r *wsE2ERepo) RemoveMember(context.Context, uuid.UUID, uuid.UUID) error       { return nil }
func (r *wsE2ERepo) GetMembers(context.Context, uuid.UUID) ([]*RoomMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.members, nil
}
func (r *wsE2ERepo) GetMember(context.Context, uuid.UUID, uuid.UUID) (*RoomMember, error) {
	return nil, nil
}
func (r *wsE2ERepo) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.room == nil || r.room.ID != roomID {
		return false, nil
	}
	for _, m := range r.members {
		if m.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}
func (r *wsE2ERepo) UpdateMemberRole(context.Context, uuid.UUID, uuid.UUID, MemberRole) error {
	return nil
}
func (r *wsE2ERepo) HasCastingResponseAccess(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (r *wsE2ERepo) CreateMessage(context.Context, *Message) error { return nil }
func (r *wsE2ERepo) GetMessageByID(context.Context, uuid.UUID) (*Message, error) {
	return nil, nil
}
func (r *wsE2ERepo) ListMessagesByRoom(context.Context, uuid.UUID, int, int) ([]*Message, error) {
	return nil, nil
}
func (r *wsE2ERepo) DeleteMessage(context.Context, uuid.UUID) error { return nil }
func (r *wsE2ERepo) MarkMessagesAsRead(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (r *wsE2ERepo) CountUnreadByRoom(context.Context, uuid.UUID, uuid.UUID) (int, error) {
	return 0, nil
}
func (r *wsE2ERepo) CountUnreadByUser(context.Context, uuid.UUID) (int, error) { return 0, nil }

type wsE2EUploadResolver struct{}

func (r *wsE2EUploadResolver) GetAttachmentInfo(context.Context, uuid.UUID, uuid.UUID) (*AttachmentInfo, error) {
	return &AttachmentInfo{
		UploadID: uuid.New(),
		URL:      "https://cdn.example.com/chat-attachment.jpg",
		FileName: "chat-attachment.jpg",
		MimeType: "image/jpeg",
		Size:     1024,
	}, nil
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func readEvent(t *testing.T, conn *websocket.Conn) WSEvent {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read ws message: %v", err)
	}
	var event WSEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		t.Fatalf("unmarshal ws event: %v (%s)", err, string(msg))
	}
	return event
}

func readEventByType(t *testing.T, conn *websocket.Conn, eventType EventType) WSEvent {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		event := readEvent(t, conn)
		if event.Type == eventType {
			return event
		}
	}
	t.Fatalf("timeout waiting event type %s", eventType)
	return WSEvent{}
}

func waitForGauge(t *testing.T, srvURL string, want int64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(srvURL + "/debug/vars")
		if err == nil {
			var vars map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&vars)
			_ = resp.Body.Close()
			v, _ := vars["websocket_connections"].(float64)
			if int64(v) == want {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	resp, _ := http.Get(srvURL + "/debug/vars")
	defer func() { _ = resp.Body.Close() }()
	var vars map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&vars)
	t.Fatalf("expected websocket_connections=%d got %#v", want, vars["websocket_connections"])
}

func TestWebSocketRealtimeE2E_MessageAttachmentReadAndMetrics(t *testing.T) {
	wsConnectionsGauge.Set(0)
	wsEventsSentTotal.Set(0)
	wsEventsDroppedTotal.Set(0)

	jwtService := jwtpkg.NewService("test-secret", time.Hour, 2*time.Hour)
	authMw := middleware.Auth(jwtService)

	userA := uuid.New()
	userB := uuid.New()
	roomID := uuid.New()

	repo := &wsE2ERepo{
		room: &Room{ID: roomID, RoomType: RoomTypeDirect, CreatedAt: time.Now()},
		members: []*RoomMember{
			{ID: uuid.New(), RoomID: roomID, UserID: userA, Role: MemberRoleMember, JoinedAt: time.Now()},
			{ID: uuid.New(), RoomID: roomID, UserID: userB, Role: MemberRoleMember, JoinedAt: time.Now()},
		},
	}
	users := &testUserRepo{users: map[uuid.UUID]*user.User{
		userA: {ID: userA, Email: "a@example.com"},
		userB: {ID: userB, Email: "b@example.com"},
	}}

	hub := NewHub(nil)
	go hub.Run()
	defer hub.Shutdown()

	svc := NewService(repo, users, hub, &noopAccessChecker{}, nil, &wsE2EUploadResolver{})
	h := NewHandler(svc, hub, nil, nil, nil, nil, nil)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/ws", func(w http.ResponseWriter, req *http.Request) {
		token := req.URL.Query().Get("token")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		authMw(http.HandlerFunc(h.WebSocket)).ServeHTTP(w, req)
	})
	r.Post("/api/v1/chat/rooms/{id}/messages", authMw(http.HandlerFunc(h.SendMessage)).ServeHTTP)
	r.Post("/api/v1/chat/rooms/{id}/read", authMw(http.HandlerFunc(h.MarkAsRead)).ServeHTTP)
	r.Handle("/debug/vars", expvar.Handler())

	ts := httptest.NewServer(r)
	defer ts.Close()

	tokenA, _ := jwtService.GenerateAccessToken(userA, "model", false)
	tokenB, _ := jwtService.GenerateAccessToken(userB, "model", false)

	dialer := websocket.Dialer{}
	connA, respA, err := dialer.Dial(fmt.Sprintf("%s/ws?token=%s", wsURL(ts.URL), tokenA), nil)
	if err != nil {
		t.Fatalf("ws dial A failed: %v", err)
	}
	if respA == nil || respA.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected status 101 for A, got %#v", respA)
	}
	defer connA.Close()

	connB, respB, err := dialer.Dial(fmt.Sprintf("%s/ws?token=%s", wsURL(ts.URL), tokenB), nil)
	if err != nil {
		t.Fatalf("ws dial B failed: %v", err)
	}
	if respB == nil || respB.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("expected status 101 for B, got %#v", respB)
	}
	defer connB.Close()

	waitForGauge(t, ts.URL, 2)

	join := map[string]any{"type": "join", "room_id": roomID}
	if err := connA.WriteJSON(join); err != nil {
		t.Fatalf("join A write: %v", err)
	}
	if err := connB.WriteJSON(join); err != nil {
		t.Fatalf("join B write: %v", err)
	}

	// 1) text message -> realtime new_message to B
	body := strings.NewReader(`{"content":"hello realtime","message_type":"text"}`)
	reqMsg, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/chat/rooms/"+roomID.String()+"/messages", body)
	reqMsg.Header.Set("Authorization", "Bearer "+tokenA)
	reqMsg.Header.Set("Content-Type", "application/json")
	respMsg, err := http.DefaultClient.Do(reqMsg)
	if err != nil {
		t.Fatalf("send message request: %v", err)
	}
	_ = respMsg.Body.Close()
	if respMsg.StatusCode != http.StatusCreated {
		t.Fatalf("send message status=%d", respMsg.StatusCode)
	}
	event1 := readEventByType(t, connB, EventNewMessage)
	if event1.Type != EventNewMessage {
		t.Fatalf("expected new_message, got %s", event1.Type)
	}
	if event1.Message == nil || event1.Message.Content != "hello realtime" || event1.Message.RoomID != roomID || event1.Message.SenderID != userA {
		t.Fatalf("unexpected message payload: %#v", event1.Message)
	}
	event1Created := readEventByType(t, connB, EventMessageCreate)
	if event1Created.Message == nil || event1Created.Message.Content != "hello realtime" {
		t.Fatalf("unexpected message_created payload: %#v", event1Created.Message)
	}
	event1Room := readEventByType(t, connB, EventRoomUpdated)
	if event1Room.RoomID != roomID {
		t.Fatalf("unexpected room_updated payload: %#v", event1Room)
	}

	// 2) attachment message -> realtime includes attachments[]
	uploadID := uuid.New()
	bodyAtt := strings.NewReader(fmt.Sprintf(`{"attachment_upload_ids":["%s"]}`, uploadID))
	reqAtt, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/chat/rooms/"+roomID.String()+"/messages", bodyAtt)
	reqAtt.Header.Set("Authorization", "Bearer "+tokenA)
	reqAtt.Header.Set("Content-Type", "application/json")
	respAtt, err := http.DefaultClient.Do(reqAtt)
	if err != nil {
		t.Fatalf("send attachment message request: %v", err)
	}
	_ = respAtt.Body.Close()
	if respAtt.StatusCode != http.StatusCreated {
		t.Fatalf("send attachment status=%d", respAtt.StatusCode)
	}
	event2 := readEventByType(t, connB, EventNewMessage)
	if event2.Type != EventNewMessage {
		t.Fatalf("expected attachment new_message, got %s", event2.Type)
	}
	if event2.Message == nil || len(event2.Message.Attachments) == 0 {
		t.Fatalf("expected attachments in WS payload, got %#v", event2.Message)
	}
	event2Created := readEventByType(t, connB, EventMessageCreate)
	if event2Created.Message == nil || len(event2Created.Message.Attachments) == 0 {
		t.Fatalf("expected attachments in message_created payload, got %#v", event2Created.Message)
	}
	_ = readEventByType(t, connB, EventRoomUpdated)

	// 3) read by B -> realtime read to A
	reqRead, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/chat/rooms/"+roomID.String()+"/read", nil)
	reqRead.Header.Set("Authorization", "Bearer "+tokenB)
	respRead, err := http.DefaultClient.Do(reqRead)
	if err != nil {
		t.Fatalf("mark read request: %v", err)
	}
	_ = respRead.Body.Close()
	if respRead.StatusCode != http.StatusOK {
		t.Fatalf("mark read status=%d", respRead.StatusCode)
	}
	event3 := readEventByType(t, connA, EventRead)
	if event3.Type != EventRead || event3.RoomID != roomID || event3.SenderID != userB {
		t.Fatalf("unexpected read payload: %#v", event3)
	}

	// 4) metrics check
	respVars, err := http.Get(ts.URL + "/debug/vars")
	if err != nil {
		t.Fatalf("debug vars request: %v", err)
	}
	defer respVars.Body.Close()
	if respVars.StatusCode != http.StatusOK {
		t.Fatalf("debug vars status=%d", respVars.StatusCode)
	}
	var vars map[string]any
	if err := json.NewDecoder(respVars.Body).Decode(&vars); err != nil {
		t.Fatalf("decode debug vars: %v", err)
	}
	if got := int64(vars["websocket_connections"].(float64)); got != 2 {
		t.Fatalf("expected websocket_connections=2, got %d", got)
	}
	if sent := int64(vars["websocket_events_sent_total"].(float64)); sent < 3 {
		t.Fatalf("expected websocket_events_sent_total >=3, got %d", sent)
	}
	if dropped := int64(vars["websocket_events_dropped_total"].(float64)); dropped != 0 {
		t.Fatalf("expected websocket_events_dropped_total=0, got %d", dropped)
	}
}
