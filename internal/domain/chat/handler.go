package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/domain/notification"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/errorhandler"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// WebSocket constants
const (
	writeWait                    = 10 * time.Second
	pongWait                     = 60 * time.Second
	pingPeriod                   = (pongWait * 9) / 10
	maxMessageSize               = 512 * 1024 // 512KB
	notificationSyncDefaultLimit = 20
	notificationSyncMaxLimit     = 50
	notificationSyncRateLimit    = 6
	notificationReadRateLimit    = 20
	notificationRateLimitWindow  = 10 * time.Second
)

// ProfileFetcher interface to retrieve user details
type ProfileFetcher interface {
	GetParticipantInfo(ctx context.Context, userID uuid.UUID) (*ParticipantInfo, error)
}

// NotificationQuery defines read-only notification access for WS sync
type NotificationQuery interface {
	List(ctx context.Context, userID uuid.UUID, limit, offset int, unreadOnly bool) ([]*notification.NotificationResponse, error)
	UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
}

// NotificationWriter defines user-scoped notification write operations for WS commands
type NotificationWriter interface {
	MarkAsRead(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
}

// Handler handles chat HTTP requests
type Handler struct {
	service            *Service
	hub                *Hub
	rateLimiter        *RateLimiter
	upgrader           websocket.Upgrader
	profileFetcher     ProfileFetcher
	notificationQuery  NotificationQuery
	notificationWriter NotificationWriter
	syncLimiter        *userWindowLimiter
	readLimiter        *userWindowLimiter
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

	key := fmt.Sprintf("ratelimit:chat:%s", userID) // Fixed syntax
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
func NewHandler(service *Service, hub *Hub, redisClient *redis.Client, allowedOrigins []string, profileFetcher ProfileFetcher, notificationQuery NotificationQuery, notificationWriter NotificationWriter) *Handler {
	return &Handler{
		service:            service,
		hub:                hub,
		rateLimiter:        NewRateLimiter(redisClient),
		profileFetcher:     profileFetcher,
		notificationQuery:  notificationQuery,
		notificationWriter: notificationWriter,
		syncLimiter:        newUserWindowLimiter(notificationSyncRateLimit, notificationRateLimitWindow),
		readLimiter:        newUserWindowLimiter(notificationReadRateLimit, notificationRateLimitWindow),
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
// @Description Создает комнату. Типы комнат (room_type):
// @Description - direct: личный чат (требует recipient_id)
// @Description - casting: чат кастинга (требует casting_id)
// @Description - group: групповой чат (требует name, member_ids)
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
		errorhandler.HandleError(r.Context(), w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", err)
		return
	}

	if validationErrors := validator.Validate(&req); validationErrors != nil {
		errorhandler.LogValidationError(r.Context(), validationErrors)
		response.ErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", validationErrors)
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
			errorhandler.HandleError(r.Context(), w, http.StatusBadRequest, "INVALID_REQUEST", "Cannot start chat with yourself", err)
		case ErrUserNotFound:
			errorhandler.HandleError(r.Context(), w, http.StatusNotFound, "USER_NOT_FOUND", "User not found", err)
		case ErrUserBlocked:
			errorhandler.HandleError(r.Context(), w, http.StatusForbidden, "USER_BLOCKED", "Cannot create chat - user is blocked", err)
		case ErrEmployerNotVerified:
			errorhandler.HandleError(r.Context(), w, http.StatusForbidden, "EMPLOYER_NOT_VERIFIED", "Employer account is pending verification", err)
		case ErrInvalidMembersCount:
			errorhandler.HandleError(r.Context(), w, http.StatusBadRequest, "INVALID_MEMBERS_COUNT", "At least one member is required", err)
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"Failed to create room",
				err)
		}
		return
	}

	unread, _ := h.service.repo.CountUnreadByRoom(r.Context(), room.ID, userID)
	response.Created(w, h.getRoomResponse(r.Context(), room, userID, unread))
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
		errorhandler.HandleError(r.Context(), w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"Failed to list rooms",
			err)
		return
	}

	items := make([]*RoomResponse, len(rooms))
	for i, roomItem := range rooms {
		items[i] = h.getRoomResponse(r.Context(), roomItem.Room, userID, roomItem.UnreadCount)
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
		errorhandler.HandleError(r.Context(), w, http.StatusBadRequest, "INVALID_ID", "Invalid room ID", err)
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
			errorhandler.HandleError(r.Context(), w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room not found", err)
		case ErrNotRoomMember:
			errorhandler.HandleError(r.Context(), w, http.StatusForbidden, "NOT_ROOM_MEMBER", "You are not a member of this chat", err)
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"Failed to get messages",
				err)
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
// @Description Отправка сообщения в комнату. Для файлов используйте attachment_upload_id (должен быть committed с purpose=chat_file).
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
		errorhandler.HandleError(r.Context(), w, http.StatusBadRequest, "INVALID_ID", "Invalid room ID", err)
		return
	}

	userID := middleware.GetUserID(r.Context())

	// Rate limiting
	if !h.rateLimiter.Allow(userID) {
		errorhandler.HandleError(r.Context(), w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many messages, please slow down", nil)
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorhandler.HandleError(r.Context(), w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body", err)
		return
	}

	if validationErrors := validator.Validate(&req); validationErrors != nil {
		errorhandler.LogValidationError(r.Context(), validationErrors)
		response.ErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", validationErrors)
		return
	}

	msg, err := h.service.SendMessage(r.Context(), userID, roomID, &req)
	if err != nil {
		if middleware.WriteLimitExceeded(w, err) {
			return
		}
		switch err {
		case ErrRoomNotFound:
			errorhandler.HandleError(r.Context(), w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room not found", err)
		case ErrNotRoomMember:
			errorhandler.HandleError(r.Context(), w, http.StatusForbidden, "NOT_ROOM_MEMBER", "You are not a member of this chat", err)
		case ErrUserBlocked:
			errorhandler.HandleError(r.Context(), w, http.StatusForbidden, "USER_BLOCKED", "Cannot send message - user is blocked", err)
		case ErrInvalidImageURL:
			errorhandler.HandleError(r.Context(), w, http.StatusBadRequest, "INVALID_IMAGE_URL", "Invalid image URL - must be a valid HTTP(S) URL", err)
		case ErrEmployerNotVerified:
			errorhandler.HandleError(r.Context(), w, http.StatusForbidden, "EMPLOYER_NOT_VERIFIED", "Employer account is pending verification", err)
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"Failed to send message",
				err)
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
		errorhandler.HandleError(r.Context(), w, http.StatusBadRequest, "INVALID_ID", "Invalid room ID", err)
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.MarkAsRead(r.Context(), userID, roomID); err != nil {
		switch err {
		case ErrRoomNotFound:
			errorhandler.HandleError(r.Context(), w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room not found", err)
		case ErrNotRoomMember:
			errorhandler.HandleError(r.Context(), w, http.StatusForbidden, "NOT_ROOM_MEMBER", "You are not a member of this chat", err)
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"Failed to mark as read",
				err)
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
		errorhandler.HandleError(r.Context(), w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
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

	h.sendInitialNotificationSync(userID, client)

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
		case "notification:sync":
			h.processNotificationSyncCommand(client, event.Data)
		case "notification:read":
			h.processNotificationReadCommand(client, event.Data)
		case "notification:read-all":
			h.processNotificationReadAllCommand(client)
		}
	}
}

func (h *Handler) processNotificationSyncCommand(client *Connection, raw json.RawMessage) {
	if h.syncLimiter != nil && !h.syncLimiter.Allow(client.UserID.String()) {
		log.Warn().Str("user_id", client.UserID.String()).Str("action", "notification:sync").Str("result", "rate_limited").Msg("Notification WS command rate limited")
		h.sendWSError(client, "notification_rate_limited")
		return
	}
	h.handleNotificationSyncRequest(client, raw)
}

func (h *Handler) processNotificationReadCommand(client *Connection, raw json.RawMessage) {
	if h.readLimiter != nil && !h.readLimiter.Allow(client.UserID.String()) {
		log.Warn().Str("user_id", client.UserID.String()).Str("action", "notification:read").Str("result", "rate_limited").Msg("Notification WS command rate limited")
		h.sendWSError(client, "notification_rate_limited")
		return
	}
	h.handleNotificationReadRequest(client, raw)
}

func (h *Handler) processNotificationReadAllCommand(client *Connection) {
	if h.readLimiter != nil && !h.readLimiter.Allow(client.UserID.String()) {
		log.Warn().Str("user_id", client.UserID.String()).Str("action", "notification:read-all").Str("result", "rate_limited").Msg("Notification WS command rate limited")
		h.sendWSError(client, "notification_rate_limited")
		return
	}
	h.handleNotificationReadAllRequest(client)
}

type notificationReadRequest struct {
	ID string `json:"id"`
}

func (h *Handler) handleNotificationReadRequest(client *Connection, raw json.RawMessage) {
	if h.notificationWriter == nil {
		return
	}

	var req notificationReadRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		log.Warn().Str("user_id", client.UserID.String()).Str("action", "notification:read").Str("result", "invalid_payload").Msg("Notification WS read rejected")
		h.sendWSError(client, "notification_invalid_payload")
		return
	}

	notificationID, err := uuid.Parse(req.ID)
	if err != nil {
		log.Warn().Str("user_id", client.UserID.String()).Str("action", "notification:read").Str("result", "invalid_payload").Msg("Notification WS read rejected")
		h.sendWSError(client, "notification_invalid_payload")
		return
	}

	err = h.notificationWriter.MarkAsRead(context.Background(), client.UserID, notificationID)
	if err != nil {
		if errors.Is(err, notification.ErrNotificationNotFound) {
			log.Warn().Str("user_id", client.UserID.String()).Str("notification_id", notificationID.String()).Str("action", "notification:read").Str("result", "not_found").Msg("Notification WS read not found")
			h.sendWSError(client, "notification_not_found")
			return
		}
		log.Error().Err(err).Str("user_id", client.UserID.String()).Str("notification_id", notificationID.String()).Str("action", "notification:read").Str("result", "failed").Msg("Notification WS read failed")
		h.sendWSError(client, "notification_read_failed")
		return
	}

	log.Debug().Str("user_id", client.UserID.String()).Str("notification_id", notificationID.String()).Str("action", "notification:read").Str("result", "ok").Msg("Notification WS read served")
	h.broadcastNotificationStateRead(client.UserID, []uuid.UUID{notificationID}, false)
}

func (h *Handler) handleNotificationReadAllRequest(client *Connection) {
	if h.notificationWriter == nil {
		return
	}

	if err := h.notificationWriter.MarkAllRead(context.Background(), client.UserID); err != nil {
		log.Error().Err(err).Str("user_id", client.UserID.String()).Str("action", "notification:read-all").Str("result", "failed").Msg("Notification WS read-all failed")
		h.sendWSError(client, "notification_read_failed")
		return
	}

	log.Debug().Str("user_id", client.UserID.String()).Str("action", "notification:read-all").Str("result", "ok").Msg("Notification WS read-all served")
	h.broadcastNotificationStateRead(client.UserID, nil, true)
}

func (h *Handler) broadcastNotificationStateRead(userID uuid.UUID, readIDs []uuid.UUID, readAll bool) {
	unreadCount, err := h.notificationWriter.UnreadCount(context.Background(), userID)
	if err != nil {
		return
	}

	data := map[string]interface{}{
		"unread_count": unreadCount,
	}
	if readAll {
		data["read_all"] = true
	} else {
		ids := make([]string, 0, len(readIDs))
		for _, id := range readIDs {
			ids = append(ids, id.String())
		}
		data["read_ids"] = ids
	}

	_ = h.hub.SendToUserJSON(userID, map[string]interface{}{
		"type": "notification:state",
		"data": data,
	})
}

type notificationSyncRequest struct {
	Limit      *int `json:"limit"`
	Offset     *int `json:"offset"`
	UnreadOnly bool `json:"unread_only"`
}

func (h *Handler) sendInitialNotificationSync(userID uuid.UUID, client *Connection) {
	h.sendNotificationSync(context.Background(), userID, client, notificationSyncDefaultLimit, 0, false)
}

func (h *Handler) handleNotificationSyncRequest(client *Connection, raw json.RawMessage) {
	limit := notificationSyncDefaultLimit
	offset := 0
	unreadOnly := false

	if len(raw) > 0 {
		var req notificationSyncRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			log.Warn().Str("user_id", client.UserID.String()).Str("action", "notification:sync").Str("result", "invalid_payload").Msg("Notification WS sync rejected")
			h.sendWSError(client, "notification_invalid_payload")
			return
		}
		if req.Limit != nil {
			limit = *req.Limit
		}
		if req.Offset != nil {
			offset = *req.Offset
		}
		unreadOnly = req.UnreadOnly
	}

	if limit <= 0 {
		limit = notificationSyncDefaultLimit
	}
	if limit > notificationSyncMaxLimit {
		limit = notificationSyncMaxLimit
	}
	if offset < 0 {
		offset = 0
	}

	h.sendNotificationSync(context.Background(), client.UserID, client, limit, offset, unreadOnly)
}

func (h *Handler) sendNotificationSync(ctx context.Context, userID uuid.UUID, client *Connection, limit, offset int, unreadOnly bool) {
	if h.notificationQuery == nil {
		return
	}

	unreadCount, err := h.notificationQuery.UnreadCount(ctx, userID)
	if err != nil {
		h.sendWSError(client, "notification_sync_failed")
		return
	}

	items, err := h.notificationQuery.List(ctx, userID, limit, offset, unreadOnly)
	if err != nil {
		h.sendWSError(client, "notification_sync_failed")
		return
	}

	log.Debug().Str("user_id", userID.String()).Int("limit", limit).Int("offset", offset).Bool("unread_only", unreadOnly).Str("action", "notification:sync").Str("result", "ok").Msg("Notification WS sync served")

	h.sendWSJSON(client, map[string]interface{}{
		"type": "notification:sync",
		"data": map[string]interface{}{
			"unread_count": unreadCount,
			"items":        items,
			"limit":        limit,
			"offset":       offset,
		},
	})
}

func (h *Handler) sendWSError(client *Connection, code string) {
	h.sendWSJSON(client, map[string]interface{}{
		"type": "error",
		"data": map[string]string{"code": code},
	})
}

func (h *Handler) sendWSJSON(client *Connection, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	select {
	case client.Send <- data:
	default:
		log.Warn().Str("user_id", client.UserID.String()).Msg("WebSocket send buffer full")
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

func (h *Handler) getRoomResponse(ctx context.Context, room *Room, currentUserID uuid.UUID, unreadCount int) *RoomResponse {
	// Get members
	members, err := h.service.GetMembers(ctx, currentUserID, room.ID)
	if err != nil {
		// Fallback
		members = []*RoomMember{}
	}

	// Convert to ParticipantInfo
	participantInfos := make([]ParticipantInfo, len(members))
	var isAdmin bool
	for i, m := range members {
		info, err := h.profileFetcher.GetParticipantInfo(ctx, m.UserID)
		if err == nil {
			participantInfos[i] = *info
		} else {
			participantInfos[i] = ParticipantInfo{ID: m.UserID, FirstName: "Unknown"}
		}

		if m.UserID == currentUserID && m.IsAdmin() {
			isAdmin = true
		}
	}

	return RoomResponseFromEntity(room, participantInfos, isAdmin, unreadCount)
}

// GetMembers handles GET /chat/rooms/{id}/members
// @Summary Получить список участников комнаты
// @Description Возвращает список участников с их профилями (имя, аватар).
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID комнаты"
// @Success 200 {object} response.Response{data=[]ParticipantInfo}
// @Failure 400,403,404,500 {object} response.Response
// @Router /chat/rooms/{id}/members [get]
func (h *Handler) GetMembers(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid room ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	members, err := h.service.GetMembers(r.Context(), userID, roomID)
	if err != nil {
		switch err {
		case ErrRoomNotFound:
			errorhandler.HandleError(r.Context(), w, http.StatusNotFound, "ROOM_NOT_FOUND", "Room not found", err)
		case ErrNotRoomMember:
			errorhandler.HandleError(r.Context(), w, http.StatusForbidden, "NOT_ROOM_MEMBER", "You are not a member of this chat", err)
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"Failed to get members",
				err)
		}
		return
	}

	// Enrich with profile data
	items := make([]ParticipantInfo, len(members))
	for i, m := range members {
		info, err := h.profileFetcher.GetParticipantInfo(r.Context(), m.UserID)
		if err == nil {
			items[i] = *info
		} else {
			items[i] = ParticipantInfo{ID: m.UserID, FirstName: "Unknown"}
		}
	}

	response.OK(w, items)
}

// AddMember handles POST /chat/rooms/{id}/members
// @Summary Добавить участника в комнату
// @Description Добавить пользователя в групповую или кастинг-комнату. Только администратор может добавлять участников.
// @Tags Chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID комнаты"
// @Param request body AddMemberRequest true "ID пользователя для добавления"
// @Success 200 {object} response.Response
// @Failure 400,403,404,422,500 {object} response.Response
// @Router /chat/rooms/{id}/members [post]
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid room ID")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.AddMember(r.Context(), userID, roomID, req.UserID); err != nil {
		switch err {
		case ErrRoomNotFound:
			response.NotFound(w, "Room not found")
		case ErrNotRoomMember:
			response.Forbidden(w, "You are not a member of this chat")
		case ErrNotRoomAdmin:
			response.Forbidden(w, "Only room admin can add members")
		case ErrAlreadyMember:
			response.Error(w, http.StatusConflict, "already_member", "User is already a member")
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"Failed to add member",
				err)
		}
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// RemoveMember handles DELETE /chat/rooms/{id}/members/{userId}
// @Summary Удалить участника из комнаты
// @Description Удалить пользователя из групповой или кастинг-комнаты. Только администратор может удалять участников.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID комнаты"
// @Param userId path string true "ID пользователя для удаления"
// @Success 200 {object} response.Response
// @Failure 400,403,404,500 {object} response.Response
// @Router /chat/rooms/{id}/members/{userId} [delete]
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid room ID")
		return
	}

	targetUserID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.RemoveMember(r.Context(), userID, roomID, targetUserID); err != nil {
		switch err {
		case ErrRoomNotFound:
			response.NotFound(w, "Room not found")
		case ErrNotRoomMember:
			response.Forbidden(w, "You are not a member of this chat")
		case ErrNotRoomAdmin:
			response.Forbidden(w, "Only room admin can remove members")
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"Failed to remove member",
				err)
		}
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// LeaveRoom handles POST /chat/rooms/{id}/leave
// @Summary Покинуть комнату
// @Description Покинуть групповую или кастинг-комнату.
// @Tags Chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID комнаты"
// @Success 200 {object} response.Response
// @Failure 400,403,404,500 {object} response.Response
// @Router /chat/rooms/{id}/leave [post]
func (h *Handler) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid room ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	// Self-leave is handled by RemoveMember when userID == targetUserID
	if err := h.service.RemoveMember(r.Context(), userID, roomID, userID); err != nil {
		switch err {
		case ErrRoomNotFound:
			response.NotFound(w, "Room not found")
		case ErrNotRoomMember:
			response.Forbidden(w, "You are not a member of this chat")
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"Failed to leave room",
				err)
		}
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}
