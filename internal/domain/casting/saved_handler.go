package casting

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// SavedCastingsHandler handles saved/favorite castings HTTP requests
type SavedCastingsHandler struct {
	repo *SavedCastingRepository
}

// NewSavedCastingsHandler creates saved castings handler
func NewSavedCastingsHandler(db *sqlx.DB) *SavedCastingsHandler {
	return &SavedCastingsHandler{
		repo: NewSavedCastingRepository(db),
	}
}

// Save handles POST /castings/{id}/save
// @Summary Сохранить кастинг
// @Tags Casting
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID кастинга"
// @Success 201 {object} response.Response
// @Failure 400,401,500 {object} response.Response
// @Router /castings/{id}/save [post]
func (h *SavedCastingsHandler) Save(w http.ResponseWriter, r *http.Request) {
	castingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	err = h.repo.Save(r.Context(), userID, castingID)
	if err == ErrAlreadySaved {
		response.OK(w, map[string]interface{}{
			"saved":      true,
			"casting_id": castingID.String(),
			"message":    "Already saved",
		})
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, map[string]interface{}{
		"saved":      true,
		"casting_id": castingID.String(),
	})
}

// Unsave handles DELETE /castings/{id}/save
// @Summary Убрать кастинг из сохраненных
// @Tags Casting
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID кастинга"
// @Success 200 {object} response.Response
// @Failure 400,401,404,500 {object} response.Response
// @Router /castings/{id}/save [delete]
func (h *SavedCastingsHandler) Unsave(w http.ResponseWriter, r *http.Request) {
	castingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	err = h.repo.Unsave(r.Context(), userID, castingID)
	if err == sql.ErrNoRows {
		response.NotFound(w, "Casting not saved")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]interface{}{
		"saved":      false,
		"casting_id": castingID.String(),
	})
}

// ListSaved handles GET /castings/saved
// @Summary Список сохраненных кастингов
// @Tags Casting
// @Produce json
// @Security BearerAuth
// @Param page query int false "Страница"
// @Param limit query int false "Лимит"
// @Success 200 {object} response.Response
// @Failure 401,500 {object} response.Response
// @Router /castings/saved [get]
func (h *SavedCastingsHandler) ListSaved(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	// Pagination
	query := r.URL.Query()
	page := 1
	limit := 20
	if p := query.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if l := query.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	offset := (page - 1) * limit

	saved, err := h.repo.GetSavedCastings(r.Context(), userID, limit, offset)
	if err != nil {
		response.InternalError(w)
		return
	}

	total, _ := h.repo.CountSaved(r.Context(), userID)

	// Get casting details for saved items
	type SavedItem struct {
		CastingID string `json:"casting_id"`
		SavedAt   string `json:"saved_at"`
		// Casting details would be joined here in production
	}

	items := make([]SavedItem, len(saved))
	for i, s := range saved {
		items[i] = SavedItem{
			CastingID: s.CastingID.String(),
			SavedAt:   s.CreatedAt.Format(time.RFC3339),
		}
	}

	response.WithMeta(w, items, response.Meta{
		Total:   total,
		Page:    page,
		Limit:   limit,
		Pages:   (total + limit - 1) / limit,
		HasNext: page*limit < total,
		HasPrev: page > 1,
	})
}

// CheckSaved handles GET /castings/{id}/saved (check if saved by current user)
// @Summary Проверить, сохранен ли кастинг
// @Tags Casting
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID кастинга"
// @Success 200 {object} response.Response
// @Failure 400,500 {object} response.Response
// @Router /castings/{id}/saved [get]
func (h *SavedCastingsHandler) CheckSaved(w http.ResponseWriter, r *http.Request) {
	castingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.OK(w, map[string]bool{"saved": false})
		return
	}

	saved, err := h.repo.IsSaved(r.Context(), userID, castingID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]bool{"saved": saved})
}

// Routes returns saved castings routes
func (h *SavedCastingsHandler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/", h.ListSaved)

	return r
}
