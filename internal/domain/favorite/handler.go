package favorite

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler for favorites API
type Handler struct {
	repo *Repository
}

// NewHandler creates favorites handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// AddRequest for adding a favorite
type AddRequest struct {
	EntityType string `json:"entity_type" validate:"required,oneof=casting profile"`
	EntityID   string `json:"entity_id" validate:"required,uuid"`
}

// Add handles POST /favorites
// @Summary Добавить в избранное
// @Tags Favorite
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AddRequest true "Данные избранного"
// @Success 201 {object} response.Response
// @Failure 400,401,500 {object} response.Response
// @Router /favorites [post]
func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req AddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	entityID, err := uuid.Parse(req.EntityID)
	if err != nil {
		response.BadRequest(w, "invalid entity_id")
		return
	}

	entityType := EntityType(req.EntityType)
	if entityType != EntityTypeCasting && entityType != EntityTypeProfile {
		response.BadRequest(w, "entity_type must be 'casting' or 'profile'")
		return
	}

	fav, err := h.repo.Add(r.Context(), userID, entityType, entityID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, fav)
}

// Remove handles DELETE /favorites/:type/:id
// @Summary Удалить из избранного
// @Tags Favorite
// @Produce json
// @Security BearerAuth
// @Param type path string true "Тип сущности (casting|profile)"
// @Param id path string true "ID сущности"
// @Success 204 {string} string "No Content"
// @Failure 400,401,500 {object} response.Response
// @Router /favorites/{type}/{id} [delete]
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	entityType := EntityType(chi.URLParam(r, "type"))
	entityIDStr := chi.URLParam(r, "id")

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		response.BadRequest(w, "invalid entity_id")
		return
	}

	if err := h.repo.Remove(r.Context(), userID, entityType, entityID); err != nil {
		response.InternalError(w)
		return
	}

	response.NoContent(w)
}

// @Summary Проверить наличие в избранном
// @Tags Favorite
// @Produce json
// @Security BearerAuth
// @Param type path string true "Тип сущности (casting|profile)"
// @Param id path string true "ID сущности"
// @Success 200 {object} response.Response{data=map[string]bool}
// @Failure 400,401,500 {object} response.Response
// @Router /favorites/{type}/{id}/check [get]
// Check handles GET /favorites/:type/:id/check
func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	entityType := EntityType(chi.URLParam(r, "type"))
	entityIDStr := chi.URLParam(r, "id")

	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		response.BadRequest(w, "invalid entity_id")
		return
	}

	isFavorited, _ := h.repo.IsFavorited(r.Context(), userID, entityType, entityID)

	response.OK(w, map[string]bool{"is_favorited": isFavorited})
}

// @Summary Список избранного
// @Tags Favorite
// @Produce json
// @Security BearerAuth
// @Param type query string false "Тип фильтра (casting|profile)"
// @Param limit query int false "Лимит"
// @Param offset query int false "Смещение"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 401,500 {object} response.Response
// @Router /favorites [get]
// List handles GET /favorites
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	// Optional type filter
	var entityType *EntityType
	if t := r.URL.Query().Get("type"); t != "" {
		et := EntityType(t)
		entityType = &et
	}

	// Pagination
	limit := 20
	offset := 0
	// Parse from query params if needed

	favorites, total, err := h.repo.ListByUser(r.Context(), userID, entityType, limit, offset)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]interface{}{
		"items": favorites,
		"total": total,
	})
}

// Routes returns favorites routes
func Routes(h *Handler, authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Post("/", h.Add)
	r.Get("/", h.List)
	r.Delete("/{type}/{id}", h.Remove)
	r.Get("/{type}/{id}/check", h.Check)

	return r
}
