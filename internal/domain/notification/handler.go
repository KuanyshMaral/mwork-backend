package notification

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler handles notification HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates notification handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List handles GET /notifications
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	limit := 20
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

	notifications, err := h.service.List(r.Context(), userID, limit, offset)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*NotificationResponse, len(notifications))
	for i, n := range notifications {
		items[i] = NotificationResponseFromEntity(n)
	}

	response.OK(w, items)
}

// GetUnreadCount handles GET /notifications/unread-count
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	count, _ := h.service.GetUnreadCount(r.Context(), userID)
	response.OK(w, UnreadCountResponse{UnreadCount: count})
}

// MarkAsRead handles POST /notifications/{id}/read
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid notification ID")
		return
	}

	if err := h.service.MarkAsRead(r.Context(), id); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// MarkAllAsRead handles POST /notifications/read-all
func (h *Handler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if err := h.service.MarkAllAsRead(r.Context(), userID); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// Routes returns notification router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/", h.List)
	r.Get("/unread-count", h.GetUnreadCount)
	r.Get("/unread", h.GetUnreadCount) // Alias for frontend compatibility
	r.Post("/{id}/read", h.MarkAsRead)
	r.Post("/read-all", h.MarkAllAsRead)

	return r
}
