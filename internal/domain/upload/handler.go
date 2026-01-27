package upload

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/storage"
)

const MaxUploadSize = 20 * 1024 * 1024 // 20 MB

// Handler handles upload HTTP requests
type Handler struct {
	service        *Service
	stagingBaseURL string
}

// NewHandler creates upload handler
func NewHandler(service *Service, stagingBaseURL string) *Handler {
	return &Handler{
		service:        service,
		stagingBaseURL: stagingBaseURL,
	}
}

// Stage handles POST /uploads/stage
// Multipart form: file + category
func (h *Handler) Stage(w http.ResponseWriter, r *http.Request) {
	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		response.BadRequest(w, "File too large or invalid form")
		return
	}

	// Get category from form
	category := r.FormValue("category")
	if category == "" {
		category = "photo" // Default
	}
	if category != "avatar" && category != "photo" && category != "document" {
		response.BadRequest(w, "Invalid category. Must be: avatar, photo, or document")
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "No file provided")
		return
	}
	defer file.Close()

	userID := middleware.GetUserID(r.Context())

	// Stage the file
	upload, err := h.service.Stage(r.Context(), userID, Category(category), header.Filename, file)
	if err != nil {
		switch err {
		case storage.ErrFileTooLarge:
			response.BadRequest(w, "File exceeds maximum size")
		case storage.ErrInvalidMimeType:
			response.BadRequest(w, "File type not allowed")
		case storage.ErrEmptyFile:
			response.BadRequest(w, "File is empty")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, UploadResponseFromEntity(upload, h.stagingBaseURL))
}

// Commit handles POST /uploads/{id}/commit
func (h *Handler) Commit(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid upload ID")
		return
	}

	userID := middleware.GetUserID(r.Context())

	upload, err := h.service.Commit(r.Context(), id, userID)
	if err != nil {
		switch err {
		case ErrUploadNotFound:
			response.NotFound(w, "Upload not found")
		case ErrNotUploadOwner:
			response.Forbidden(w, "Not upload owner")
		case ErrAlreadyCommitted:
			response.BadRequest(w, "Upload already committed")
		case ErrUploadExpired:
			response.BadRequest(w, "Upload has expired, please upload again")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, UploadResponseFromEntity(upload, h.stagingBaseURL))
}

// GetByID handles GET /uploads/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid upload ID")
		return
	}

	upload, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "Upload not found")
		return
	}

	response.OK(w, UploadResponseFromEntity(upload, h.stagingBaseURL))
}

// Delete handles DELETE /uploads/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid upload ID")
		return
	}

	userID := middleware.GetUserID(r.Context())

	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		switch err {
		case ErrUploadNotFound:
			response.NotFound(w, "Upload not found")
		case ErrNotUploadOwner:
			response.Forbidden(w, "Not upload owner")
		default:
			response.InternalError(w)
		}
		return
	}

	response.NoContent(w)
}

// ListMy handles GET /uploads
func (h *Handler) ListMy(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	category := Category(r.URL.Query().Get("category"))

	uploads, err := h.service.ListByUser(r.Context(), userID, category)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*UploadResponse, len(uploads))
	for i, u := range uploads {
		items[i] = UploadResponseFromEntity(u, h.stagingBaseURL)
	}

	response.OK(w, items)
}
