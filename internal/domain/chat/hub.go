package chat

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// EventType for WebSocket messages
type EventType string

const (
	EventNewMessage    EventType = "new_message"
	EventTyping        EventType = "typing"
	EventRead          EventType = "read"
	EventOnline        EventType = "online"
	EventOffline       EventType = "offline"
	EventDeleteMessage EventType = "message_deleted"
)

// Redis key prefixes
const (
	roomChannelPrefix = "chat:room:"
	presenceKey       = "chat:presence:online"
	presenceChannel   = "chat:presence"
	userEventsChannel = "ws:user_events"
)

var (
	wsConnectionsGauge   = expvar.NewInt("websocket_connections")
	wsEventsSentTotal    = expvar.NewInt("websocket_events_sent_total")
	wsEventsDroppedTotal = expvar.NewInt("websocket_events_dropped_total")
)

type userEventMessage struct {
	EventType        string          `json:"event_type"`
	UserID           string          `json:"user_id"`
	Payload          json.RawMessage `json:"payload"`
	SenderInstanceID string          `json:"sender_instance_id"`
}

// WSEvent represents a WebSocket event
type WSEvent struct {
	Type      EventType   `json:"type"`
	RoomID    uuid.UUID   `json:"room_id,omitempty"`
	SenderID  uuid.UUID   `json:"sender_id,omitempty"`
	MessageID uuid.UUID   `json:"message_id,omitempty"`
	Message   *Message    `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// Connection represents a WebSocket connection
type Connection struct {
	UserID uuid.UUID
	Conn   *websocket.Conn
	Send   chan []byte
}

// Hub manages WebSocket connections with Redis Pub/Sub for scalability
type Hub struct {
	// Local connections (this server instance only)
	connections map[uuid.UUID]map[*Connection]bool

	// Local room subscriptions: roomID -> set of userIDs on this server
	localRooms map[uuid.UUID]map[uuid.UUID]bool

	// Redis client for Pub/Sub
	redis *redis.Client

	// Redis Pub/Sub subscription
	pubsub *redis.PubSub

	mu sync.RWMutex

	// Channels for connection management
	register   chan *Connection
	unregister chan *Connection

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	instanceID         string
	publishUserEventFn func(ctx context.Context, channel string, payload []byte) error
}

// NewHub creates a new WebSocket hub with Redis Pub/Sub
func NewHub(redisClient *redis.Client) *Hub {
	return NewHubWithInstanceID(redisClient, uuid.NewString())
}

// NewHubWithInstanceID creates a new WebSocket hub with explicit instance identifier.
func NewHubWithInstanceID(redisClient *redis.Client, instanceID string) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	h := &Hub{
		connections: make(map[uuid.UUID]map[*Connection]bool),
		localRooms:  make(map[uuid.UUID]map[uuid.UUID]bool),
		redis:       redisClient,
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		ctx:         ctx,
		cancel:      cancel,
		instanceID:  instanceID,
	}

	// Subscribe to presence channel and room pattern
	if redisClient != nil {
		h.pubsub = redisClient.PSubscribe(ctx, roomChannelPrefix+"*", presenceChannel, userEventsChannel)
		h.publishUserEventFn = func(ctx context.Context, channel string, payload []byte) error {
			return redisClient.Publish(ctx, channel, payload).Err()
		}
	}

	return h
}

// Run starts the hub (call in goroutine)
func (h *Hub) Run() {
	// Start Redis subscriber in separate goroutine
	if h.pubsub != nil {
		go h.runRedisSubscriber()
	}

	for {
		select {
		case <-h.ctx.Done():
			return

		case conn := <-h.register:
			h.mu.Lock()
			if h.connections[conn.UserID] == nil {
				h.connections[conn.UserID] = make(map[*Connection]bool)
			}
			h.connections[conn.UserID][conn] = true
			h.mu.Unlock()
			wsConnectionsGauge.Add(1)

			// Publish online status to all servers
			h.publishPresence(conn.UserID, true)
			log.Debug().Str("user_id", conn.UserID.String()).Msg("User connected to WebSocket")

		case conn := <-h.unregister:
			shouldPublishOffline := false
			h.mu.Lock()
			if conns, ok := h.connections[conn.UserID]; ok {
				if _, exists := conns[conn]; exists {
					delete(conns, conn)
					close(conn.Send)
					wsConnectionsGauge.Add(-1)
				}
				if len(conns) == 0 {
					delete(h.connections, conn.UserID)
					shouldPublishOffline = true
				}

				// Remove from all local rooms
				for roomID, users := range h.localRooms {
					delete(users, conn.UserID)
					if len(users) == 0 {
						delete(h.localRooms, roomID)
					}
				}
			}
			h.mu.Unlock()

			// Publish offline status to all servers
			if shouldPublishOffline {
				h.publishPresence(conn.UserID, false)
			}
			log.Debug().Str("user_id", conn.UserID.String()).Msg("User disconnected from WebSocket")
		}
	}
}

// runRedisSubscriber listens for messages from Redis Pub/Sub
func (h *Hub) runRedisSubscriber() {
	ch := h.pubsub.Channel()

	for {
		select {
		case <-h.ctx.Done():
			return

		case msg, ok := <-ch:
			if !ok {
				return
			}

			// Handle room messages: chat:room:<uuid>
			if len(msg.Channel) > len(roomChannelPrefix) &&
				msg.Channel[:len(roomChannelPrefix)] == roomChannelPrefix {

				roomIDStr := msg.Channel[len(roomChannelPrefix):]
				roomID, err := uuid.Parse(roomIDStr)
				if err != nil {
					continue
				}

				var event WSEvent
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					continue
				}

				// Broadcast to local clients only
				h.broadcastLocal(roomID, &event)
			}

			// Handle presence: chat:presence
			if msg.Channel == presenceChannel {
				// Just log for now, can be extended for presence UI
				log.Debug().Str("presence", msg.Payload).Msg("Presence update received")
			}

			if msg.Channel == userEventsChannel {
				h.handleUserEventPayload(msg.Payload)
			}
		}
	}
}

func (h *Hub) handleUserEventPayload(payload string) {
	var event userEventMessage
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return
	}
	if event.SenderInstanceID == h.instanceID {
		return
	}
	userID, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}
	h.sendLocalToUserJSON(userID, []byte(event.Payload))
}

// broadcastLocal sends event to clients connected to THIS server
func (h *Hub) broadcastLocal(roomID uuid.UUID, event *WSEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	users, ok := h.localRooms[roomID]
	if !ok {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	for userID := range users {
		if conns, ok := h.connections[userID]; ok {
			for conn := range conns {
				select {
				case conn.Send <- data:
					wsEventsSentTotal.Add(1)
				default:
					// Buffer full, skip this message
					wsEventsDroppedTotal.Add(1)
					log.Warn().Str("user_id", userID.String()).Msg("WebSocket send buffer full")
				}
			}
		}
	}
}

// Register adds a connection
func (h *Hub) Register(conn *Connection) {
	h.register <- conn
}

// Unregister removes a connection
func (h *Hub) Unregister(conn *Connection) {
	h.unregister <- conn
}

// SubscribeToRoom adds user to room (local subscription + notifies other servers)
func (h *Hub) SubscribeToRoom(roomID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.localRooms[roomID] == nil {
		h.localRooms[roomID] = make(map[uuid.UUID]bool)
	}
	h.localRooms[roomID][userID] = true
}

// UnsubscribeFromRoom removes user from room
func (h *Hub) UnsubscribeFromRoom(roomID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.localRooms[roomID] != nil {
		delete(h.localRooms[roomID], userID)
		if len(h.localRooms[roomID]) == 0 {
			delete(h.localRooms, roomID)
		}
	}
}

// BroadcastToRoom sends event to ALL users in room across ALL servers via Redis
func (h *Hub) BroadcastToRoom(roomID uuid.UUID, event *WSEvent) {
	if event != nil {
		log.Debug().Str("room_id", roomID.String()).Str("event_type", string(event.Type)).Msg("Broadcasting WebSocket event")
	}
	data, err := json.Marshal(event)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal WebSocket event")
		return
	}

	channel := roomChannelPrefix + roomID.String()

	if h.redis != nil {
		// Publish to Redis - all servers will receive this
		err = h.redis.Publish(h.ctx, channel, data).Err()
		if err != nil {
			log.Error().Err(err).Str("channel", channel).Msg("Redis publish failed")
			// Fallback to local broadcast
			h.broadcastLocal(roomID, event)
		}
	} else {
		// No Redis, broadcast locally only
		h.broadcastLocal(roomID, event)
	}
}

// SendToUser sends event to specific user (on any server)
func (h *Hub) SendToUser(userID uuid.UUID, event *WSEvent) {
	_ = h.SendToUserJSON(userID, event)
}

// SendToUserJSON sends JSON payload to all active connections for user.
func (h *Hub) SendToUserJSON(userID uuid.UUID, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	h.sendLocalToUserJSON(userID, data)
	if err := h.publishUserEvent(userID, data); err != nil {
		return err
	}

	return nil
}

func (h *Hub) sendLocalToUserJSON(userID uuid.UUID, data []byte) {
	h.mu.RLock()
	conns, ok := h.connections[userID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for conn := range conns {
		select {
		case conn.Send <- data:
			wsEventsSentTotal.Add(1)
		default:
			// Buffer full
			wsEventsDroppedTotal.Add(1)
		}
	}

	return
}

func (h *Hub) publishUserEvent(userID uuid.UUID, data []byte) error {
	if h.publishUserEventFn == nil {
		return nil
	}

	event := userEventMessage{
		EventType:        "notification:new",
		UserID:           userID.String(),
		Payload:          data,
		SenderInstanceID: h.instanceID,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return h.publishUserEventFn(h.ctx, userEventsChannel, payload)
}

// publishPresence publishes user online/offline status to Redis
func (h *Hub) publishPresence(userID uuid.UUID, online bool) {
	if h.redis == nil {
		return
	}

	ctx := context.Background()

	if online {
		// Add to presence set
		h.redis.SAdd(ctx, presenceKey, userID.String())
		// Set expiry for auto-cleanup (5 minutes)
		h.redis.Expire(ctx, presenceKey, 5*time.Minute)
		// Publish presence event
		h.redis.Publish(ctx, presenceChannel, fmt.Sprintf("%s:online", userID))
	} else {
		// Remove from presence set
		h.redis.SRem(ctx, presenceKey, userID.String())
		// Publish presence event
		h.redis.Publish(ctx, presenceChannel, fmt.Sprintf("%s:offline", userID))
	}
}

// IsOnline checks if user is online (across all servers)
func (h *Hub) IsOnline(userID uuid.UUID) bool {
	if h.redis == nil {
		// Check local only
		h.mu.RLock()
		conns, ok := h.connections[userID]
		h.mu.RUnlock()
		return ok && len(conns) > 0
	}

	return h.redis.SIsMember(context.Background(), presenceKey, userID.String()).Val()
}

// GetOnlineUsers returns list of online users from given list
func (h *Hub) GetOnlineUsers(userIDs []uuid.UUID) []uuid.UUID {
	if h.redis == nil {
		// Check local only
		h.mu.RLock()
		defer h.mu.RUnlock()

		online := make([]uuid.UUID, 0)
		for _, id := range userIDs {
			if conns, ok := h.connections[id]; ok && len(conns) > 0 {
				online = append(online, id)
			}
		}
		return online
	}

	// Check Redis
	members := h.redis.SMembers(context.Background(), presenceKey).Val()
	memberSet := make(map[string]bool)
	for _, m := range members {
		memberSet[m] = true
	}

	online := make([]uuid.UUID, 0)
	for _, id := range userIDs {
		if memberSet[id.String()] {
			online = append(online, id)
		}
	}

	return online
}

// GetConnectionCount returns number of local connections
func (h *Hub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, conns := range h.connections {
		total += len(conns)
	}
	return total
}

// LocalRoomUserCount returns number of users subscribed locally to room.
func (h *Hub) LocalRoomUserCount(roomID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.localRooms[roomID])
}

// IsUserSubscribedToRoom reports whether user is subscribed locally to room.
func (h *Hub) IsUserSubscribedToRoom(roomID, userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	users := h.localRooms[roomID]
	if users == nil {
		return false
	}
	return users[userID]
}

// Shutdown gracefully shuts down the hub
func (h *Hub) Shutdown() {
	h.cancel()
	if h.pubsub != nil {
		h.pubsub.Close()
	}
}
