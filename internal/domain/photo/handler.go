package photo

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles photo HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates photo handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Presign handles POST /uploads/presign
// @Summary Получить presigned URL для загрузки фото
// @Tags Photo
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body PresignRequest true "Параметры загрузки"
// @Success 200 {object} response.Response
// @Failure 400,403,500 {object} response.Response
// @Router /uploads/presign [post]
func (h *Handler) Presign(w http.ResponseWriter, r *http.Request) {
	var req PresignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	result, err := h.service.GeneratePresignedURL(r.Context(), userID, &req)
	if err != nil {
		switch err {
		case ErrNoProfileFound:
			response.BadRequest(w, "Create a profile first")
		case ErrOnlyModelsCanUpload:
			response.Forbidden(w, "Only models can upload photos")
		case ErrPhotoLimitReached:
			response.Forbidden(w, "Photo limit reached. Upgrade to Pro for unlimited.")
		default:
			response.BadRequest(w, err.Error())
		}
		return
	}

	response.OK(w, result)
}

// ConfirmUpload handles POST /photos
// @Summary Подтвердить загрузку фото
// @Tags Photo
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ConfirmUploadRequest true "Данные загрузки"
// @Success 201 {object} response.Response{data=PhotoResponse}
// @Failure 400,500 {object} response.Response
// @Router /photos [post]
func (h *Handler) ConfirmUpload(w http.ResponseWriter, r *http.Request) {
	var req ConfirmUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	photo, err := h.service.ConfirmUpload(r.Context(), userID, &req)
	if err != nil {
		switch err {
		case ErrNoProfileFound:
			response.BadRequest(w, "Create a profile first")
		case ErrUploadNotVerified:
			response.BadRequest(w, "File not found. Please upload first.")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, PhotoResponseFromEntity(photo))
}

// ListByProfile handles GET /profiles/{id}/photos
// @Summary Список фото профиля
// @Tags Photo
// @Produce json
// @Param id path string true "ID профиля"
// @Success 200 {object} response.Response{data=[]PhotoResponse}
// @Failure 400,500 {object} response.Response
// @Router /profiles/{id}/photos [get]
func (h *Handler) ListByProfile(w http.ResponseWriter, r *http.Request) {
	profileID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}

	photos, err := h.service.ListByProfile(r.Context(), profileID)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*PhotoResponse, len(photos))
	for i, p := range photos {
		items[i] = PhotoResponseFromEntity(p)
	}

	response.OK(w, items)
}

// @Summary Удалить фото
// @Tags Photo
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID фото"
// @Success 204 {string} string "No Content"
// @Failure 400,403,404,500 {object} response.Response
// @Router /photos/{id} [delete]
// Delete handles DELETE /photos/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	photoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid photo ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.Delete(r.Context(), userID, photoID); err != nil {
		switch err {
		case ErrPhotoNotFound:
			response.NotFound(w, "Photo not found")
		case ErrNotPhotoOwner:
			response.Forbidden(w, "You can only delete your own photos")
		default:
			response.InternalError(w)
		}
		return
	}

	response.NoContent(w)
}

// @Summary Установить аватар профиля
// @Tags Photo
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID фото"
// @Success 200 {object} response.Response{data=PhotoResponse}
// @Failure 400,403,404,500 {object} response.Response
// @Router /photos/{id}/avatar [patch]
// SetAvatar handles PATCH /photos/{id}/avatar
func (h *Handler) SetAvatar(w http.ResponseWriter, r *http.Request) {
	photoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid photo ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	photo, err := h.service.SetAvatar(r.Context(), userID, photoID)
	if err != nil {
		switch err {
		case ErrPhotoNotFound:
			response.NotFound(w, "Photo not found")
		case ErrNotPhotoOwner:
			response.Forbidden(w, "You can only manage your own photos")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, PhotoResponseFromEntity(photo))
}

// @Summary Изменить порядок фото
// @Tags Photo
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ReorderRequest true "Новый порядок фото"
// @Success 200 {object} response.Response
// @Failure 400,500 {object} response.Response
// @Router /photos/reorder [patch]
// Reorder handles PATCH /photos/reorder
func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req ReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.ReorderPhotos(r.Context(), userID, req.PhotoIDs); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}
