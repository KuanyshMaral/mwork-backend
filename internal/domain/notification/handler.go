package notification

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/errorhandler"
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
// @Summary Список уведомлений
// @Tags Notification
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Лимит"
// @Param offset query int false "Смещение"
// @Success 200 {object} response.Response{data=[]NotificationResponse}
// @Failure 500 {object} response.Response
// @Router /notifications [get]
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
		errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", err)
		return
	}

	items := make([]*NotificationResponse, len(notifications))
	for i, n := range notifications {
		items[i] = NotificationResponseFromEntity(n)
	}

	response.OK(w, items)
}

// GetUnreadCount handles GET /notifications/unread-count
// @Summary Количество непрочитанных уведомлений
// @Tags Notification
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=UnreadCountResponse}
// @Router /notifications/unread-count [get]
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	count, _ := h.service.GetUnreadCount(r.Context(), userID)
	response.OK(w, UnreadCountResponse{UnreadCount: count})
}

// MarkAsRead handles POST /notifications/{id}/read
// @Summary Отметить уведомление как прочитанное
// @Tags Notification
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID уведомления"
// @Success 200 {object} response.Response
// @Failure 400,500 {object} response.Response
// @Router /notifications/{id}/read [post]
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid notification ID")
		return
	}

	if err := h.service.MarkAsRead(r.Context(), id); err != nil {
		errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", err)
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// MarkAllAsRead handles POST /notifications/read-all
// @Summary Отметить все уведомления как прочитанные
// @Tags Notification
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /notifications/read-all [post]
func (h *Handler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if err := h.service.MarkAllAsRead(r.Context(), userID); err != nil {
		errorhandler.HandleError(r.Context(), w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", err)
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
