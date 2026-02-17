package relationships

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// ProfileFetcher interface to retrieve user details
type ProfileFetcher interface {
	GetUserProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
}

// UserProfile represents user profile data
type UserProfile struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	AvatarURL *string
}

// Handler handles relationship HTTP requests
type Handler struct {
	service        *Service
	profileFetcher ProfileFetcher
}

// NewHandler creates relationship handler
func NewHandler(service *Service, profileFetcher ProfileFetcher) *Handler {
	return &Handler{
		service:        service,
		profileFetcher: profileFetcher,
	}
}

// BlockUser handles POST /users/{id}/block
// @Summary Заблокировать пользователя
// @Description Заблокировать пользователя. Заблокированный пользователь не сможет отправлять сообщения.
// @Tags Relationships
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID пользователя для блокировки"
// @Success 200 {object} response.Response
// @Failure 400,500 {object} response.Response
// @Router /users/{id}/block [post]
func (h *Handler) BlockUser(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.BlockUser(r.Context(), userID, targetUserID); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// UnblockUser handles DELETE /users/{id}/block
// @Summary Разблокировать пользователя
// @Description Разблокировать ранее заблокированного пользователя.
// @Tags Relationships
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID пользователя для разблокировки"
// @Success 200 {object} response.Response
// @Failure 400,500 {object} response.Response
// @Router /users/{id}/block [delete]
func (h *Handler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.UnblockUser(r.Context(), userID, targetUserID); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}

// ListBlocked handles GET /users/me/blocked
// @Summary Список заблокированных пользователей
// @Description Получить список всех пользователей, заблокированных текущим пользователем.
// @Tags Relationships
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=[]BlockedUserResponse}
// @Failure 500 {object} response.Response
// @Router /users/me/blocked [get]
func (h *Handler) ListBlocked(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	blocks, err := h.service.ListMyBlocks(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Enrich with profile data
	items := make([]*BlockedUserResponse, 0, len(blocks))
	for _, block := range blocks {
		profile, err := h.profileFetcher.GetUserProfile(r.Context(), block.BlockedUserID)
		if err != nil {
			// Fallback to minimal data
			items = append(items, BlockRelationFromEntity(block, "Unknown", "", nil))
			continue
		}
		items = append(items, BlockRelationFromEntity(block, profile.FirstName, profile.LastName, profile.AvatarURL))
	}

	response.OK(w, items)
}
