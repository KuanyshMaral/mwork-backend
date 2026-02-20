package upload

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

const maxMultipartMemory = 32 * 1024 * 1024 // 32 MB in-memory before spilling to temp files

// Handler handles file upload HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates upload handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// uploadResponse is the API response shape for an upload.
type uploadResponse struct {
	ID           uuid.UUID `json:"id"`
	URL          string    `json:"url"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    string    `json:"created_at"`
}

func (h *Handler) toResponse(u *Upload) *uploadResponse {
	return &uploadResponse{
		ID:           u.ID,
		URL:          h.service.GetURL(u),
		OriginalName: u.OriginalName,
		MimeType:     u.MimeType,
		SizeBytes:    u.SizeBytes,
		CreatedAt:    u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Routes mounts the upload routes onto a chi router.
// All routes require authentication via the provided middleware.
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", h.Upload)       // POST /files
		r.Get("/{id}", h.Get)       // GET  /files/{id}
		r.Delete("/{id}", h.Delete) // DELETE /files/{id}
	}
}

// Upload handles POST /files
// Accepts multipart/form-data with a "file" field.
// Saves file to disk, writes metadata to DB, returns id + url.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	authorID := middleware.GetUserID(r.Context())
	if authorID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		response.BadRequest(w, "Failed to parse multipart form: "+err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "Missing 'file' field in form data")
		return
	}
	defer file.Close()

	upload, err := h.service.Upload(r.Context(), authorID, header.Filename, file)
	if err != nil {
		switch {
		case errors.Is(err, ErrFileTooLarge):
			response.Error(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "File exceeds the maximum allowed size (50 MB)")
		case errors.Is(err, ErrInvalidMime):
			response.Error(w, http.StatusUnprocessableEntity, "INVALID_FILE_TYPE", "File type is not allowed")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, h.toResponse(upload))
}

// Get handles GET /files/{id}
// Returns file metadata from DB. Does not stream the file — use the static URL.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid upload ID")
		return
	}

	upload, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUploadNotFound) {
			response.NotFound(w, "Upload not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, h.toResponse(upload))
}

// Delete handles DELETE /files/{id}
// Only the file's author can delete it.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	callerID := middleware.GetUserID(r.Context())
	if callerID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid upload ID")
		return
	}

	if err := h.service.Delete(r.Context(), id, callerID); err != nil {
		switch {
		case errors.Is(err, ErrUploadNotFound):
			response.NotFound(w, "Upload not found")
		case errors.Is(err, ErrNotOwner):
			response.Forbidden(w, "You do not own this file")
		default:
			response.InternalError(w)
		}
		return
	}

	response.NoContent(w)
}
