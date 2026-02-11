package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// WebSocket constants
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

// Handler handles chat HTTP requests
type Handler struct {
	service     *Service
	hub         *Hub
	rateLimiter *RateLimiter
	upgrader    websocket.Upgrader
}

// RateLimiter for chat messages
type RateLimiter struct {
	redis  *redis.Client
	limit  int
	window time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		limit:  30,          // 30 messages
		window: time.Minute, // per minute
	}
}

// Allow checks if user can send message
func (rl *RateLimiter) Allow(userID uuid.UUID) bool {
	if rl.redis == nil {
		return true // No Redis, allow all
	}

	key := fmt.Sprintf("ratelimit:chat:%s", userID)
	ctx := context.Background()

	count, err := rl.redis.Incr(ctx, key).Result()
	if err != nil {
		return true // Fail open
	}

	if count == 1 {
		rl.redis.Expire(ctx, key, rl.window)
	}

	return count <= int64(rl.limit)
}

// NewHandler creates chat handler
func NewHandler(service *Service, hub *Hub, redisClient *redis.Client, allowedOrigins []string) *Handler {
	return &Handler{
		service:     service,
		hub:         hub,
		rateLimiter: NewRateLimiter(redisClient),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")

				// Allow all in development
				if len(allowedOrigins) == 0 {
					return true
				}

				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}

				log.Warn().Str("origin", origin).Msg("WebSocket origin rejected")
				return false
			},
		},
	}
}

// CreateRoom handles POST /chat/rooms
// @Summary Создать или получить чат-комнату
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRoomRequest true "Данные для создания комнаты"
// @Success 201 {object} response.Response{data=RoomResponse}
// @Failure 400,403,404,422,429,500 {object} response.Response
// @Router /chat/rooms [post]
func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	room, err := h.service.CreateOrGetRoom(r.Context(), userID, &req)
	if err != nil {
		if middleware.WriteLimitExceeded(w, err) {
			return
		}
		switch err {
		case ErrCannotChatSelf:
			response.BadRequest(w, "Cannot start chat with yourself")
		case ErrUserNotFound:
			response.NotFound(w, "User not found")
		case ErrUserBlocked:
			response.Forbidden(w, "Cannot create chat - user is blocked")
		case ErrEmployerNotVerified:
			response.Forbidden(w, "Employer account is pending verification")
		default:
			response.InternalError(w)
		}
		return
	}

	unread, _ := h.service.repo.CountUnreadByRoom(r.Context(), room.ID, userID)
	response.Created(w, RoomResponseFromEntity(room, userID, unread))
}

// ListRooms handles GET /chat/rooms
// @Summary Список чат-комнат
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]RoomResponse}
// @Failure 500 {object} response.Response
// @Router /chat/rooms [get]
func (h *Handler) ListRooms(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	rooms, err := h.service.ListRooms(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*RoomResponse, len(rooms))
	for i, r := range rooms {
		items[i] = RoomResponseFromEntity(r.Room, userID, r.UnreadCount)
	}

	response.OK(w, items)
}

// GetMessages handles GET /chat/rooms/{id}/messages
// @Summary Получить сообщения комнаты
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID комнаты"
// @Param limit query int false "Лимит"
// @Param offset query int false "Смещение"
// @Success 200 {object} response.Response{data=[]MessageResponse}
// @Failure 400,403,404,500 {object} response.Response
// @Router /chat/rooms/{id}/messages [get]
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid room ID")
		return
	}

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	userID := middleware.GetUserID(r.Context())
	messages, err := h.service.GetMessages(r.Context(), userID, roomID, limit, offset)
	if err != nil {
		switch err {
		case ErrRoomNotFound:
			response.NotFound(w, "Room not found")
		case ErrNotRoomMember:
			response.Forbidden(w, "You are not a member of this chat")
		default:
			response.InternalError(w)
		}
		return
	}

	items := make([]*MessageResponse, len(messages))
	for i, m := range messages {
		items[i] = MessageResponseFromEntity(m, userID)
	}

	response.OK(w, items)
}

// SendMessage handles POST /chat/rooms/{id}/messages
// @Summary Отправить сообщение
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID комнаты"
// @Param request body SendMessageRequest true "Тело сообщения"
// @Success 201 {object} response.Response{data=MessageResponse}
// @Failure 400,403,404,422,429,500 {object} response.Response
// @Router /chat/rooms/{id}/messages [post]
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid room ID")
		return
	}

	userID := middleware.GetUserID(r.Context())

	// Rate limiting
	if !h.rateLimiter.Allow(userID) {
		response.Error(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Too many messages, please slow down")
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	msg, err := h.service.SendMessage(r.Context(), userID, roomID, &req)
	if err != nil {
		if middleware.WriteLimitExceeded(w, err) {
			return
		}
		switch err {
		case ErrRoomNotFound:
			response.NotFound(w, "Room not found")
		case ErrNotRoomMember:
			response.Forbidden(w, "You are not a member of this chat")
		case ErrUserBlocked:
			response.Forbidden(w, "Cannot send message - user is blocked")
		case ErrInvalidImageURL:
			response.BadRequest(w, "Invalid image URL - must be a valid HTTP(S) URL")
		case ErrEmployerNotVerified:
			response.Forbidden(w, "Employer account is pending verification")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, MessageResponseFromEntity(msg, userID))
}

// MarkAsRead handles POST /chat/rooms/{id}/read
// @Summary Отметить сообщения как прочитанные
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID комнаты"
// @Success 200 {object} response.Response
// @Failure 400,403,404,500 {object} response.Response
// @Router /chat/rooms/{id}/read [post]
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid room ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.MarkAsRead(r.Context(), userID, roomID); err != nil {
		switch err {
		case ErrRoomNotFound:
			response.NotFound(w, "Room not found")
		case ErrNotRoomMember:
			response.Forbidden(w, "You are not a member of this chat")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// GetUnreadCount handles GET /chat/unread
// @Summary Количество непрочитанных сообщений
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Router /chat/unread [get]
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	count, _ := h.service.GetUnreadCount(r.Context(), userID)
	response.OK(w, map[string]int{"unread_count": count})
}

// WebSocket handles WS /ws
func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}

	client := &Connection{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	h.hub.Register(client)

	// Subscribe to user's rooms
	rooms, _ := h.service.ListRooms(r.Context(), userID)
	for _, room := range rooms {
		h.hub.SubscribeToRoom(room.ID, userID)
	}

	// Start reader and writer goroutines
	go h.wsReader(client)
	go h.wsWriter(client)
}

func (h *Handler) wsReader(client *Connection) {
	defer func() {
		h.hub.Unregister(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("user_id", client.UserID.String()).Msg("WebSocket read error")
			}
			break
		}

		// Rate limiting for WebSocket messages
		if !h.rateLimiter.Allow(client.UserID) {
			continue
		}

		// Parse incoming message
		var event struct {
			Type   string          `json:"type"`
			RoomID uuid.UUID       `json:"room_id"`
			Data   json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		// Handle events
		switch event.Type {
		case "typing":
			h.hub.BroadcastToRoom(event.RoomID, &WSEvent{
				Type:     EventTyping,
				RoomID:   event.RoomID,
				SenderID: client.UserID,
			})
		case "read":
			_ = h.service.MarkAsRead(context.Background(), client.UserID, event.RoomID)
			h.hub.BroadcastToRoom(event.RoomID, &WSEvent{
				Type:     EventRead,
				RoomID:   event.RoomID,
				SenderID: client.UserID,
			})
		}
	}
}

func (h *Handler) wsWriter(client *Connection) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping for heartbeat
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
