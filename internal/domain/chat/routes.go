package chat

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns chat router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// All routes require authentication
	r.Use(authMiddleware)

	// Room operations
	r.Post("/rooms", h.CreateRoom)
	r.Get("/rooms", h.ListRooms)

	// Room messages
	r.Get("/rooms/{id}/messages", h.GetMessages)
	r.Post("/rooms/{id}/messages", h.SendMessage)
	r.Post("/rooms/{id}/read", h.MarkAsRead)

	// Room members
	r.Get("/rooms/{id}/members", h.GetMembers)
	r.Post("/rooms/{id}/members", h.AddMember)
	r.Delete("/rooms/{id}/members/{userId}", h.RemoveMember)
	r.Post("/rooms/{id}/leave", h.LeaveRoom)

	// Unread count
	r.Get("/unread", h.GetUnreadCount)

	return r
}

// WSRoute returns WebSocket route handler
func (h *Handler) WSRoute(authMiddleware func(http.Handler) http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Apply auth middleware manually for WebSocket
		// Token should be in query param: ?token=xxx
		h.WebSocket(w, r)
	}
}
