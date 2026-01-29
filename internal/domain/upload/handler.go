package upload

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/storage"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

const (
	MaxUploadSize     = 20 * 1024 * 1024 // multipart: 20 MB
	MaxInitUploadSize = 10 * 1024 * 1024 // init flow: 10 MB
	PresignExpiry     = 15 * time.Minute
)

type InitRequest struct {
	FileName    string `json:"file_name" validate:"required"`
	ContentType string `json:"content_type" validate:"required"`
	FileSize    int64  `json:"file_size" validate:"required,max=10485760"`
}
type InitResponse struct {
	UploadID  string `json:"upload_id"`
	UploadURL string `json:"upload_url"`
	ExpiresAt string `json:"expires_at"`
}

type ConfirmRequest struct {
	UploadID string `json:"upload_id" validate:"required,uuid"`
}

type ConfirmResponse struct {
	FileURL   string `json:"file_url"`
	FinalPath string `json:"final_path"`
}

type UploadStorage interface {
	GeneratePresignedPutURL(ctx context.Context, key string, expires time.Duration, contentType string) (string, error)
	Exists(ctx context.Context, key string) (bool, error)
	Move(ctx context.Context, srcKey, dstKey string) error
	GetURL(key string) string
}

// Handler handles upload HTTP requests

type Handler struct {
	service        *Service
	stagingBaseURL string

	storage UploadStorage
	repo    Repository
}

// NewHandler creates upload handler
func NewHandler(service *Service, stagingBaseURL string, st UploadStorage, repo Repository) *Handler {
	return &Handler{
		service:        service,
		stagingBaseURL: stagingBaseURL,
		storage:        st,
		repo:           repo,
	}
}

// Init handles POST /uploads/init
func (h *Handler) Init(w http.ResponseWriter, r *http.Request) {
	var req InitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.ValidationError(w, errs)
		return
	}

	if req.FileSize > MaxInitUploadSize {
		response.Error(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "file too large")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	uploadID := uuid.New()
	fileName := sanitizeFileName(req.FileName)

	stagingKey := fmt.Sprintf("uploads/staging/%s/%s/%s", userID.String(), uploadID.String(), fileName)

	uploadURL, err := h.storage.GeneratePresignedPutURL(r.Context(), stagingKey, PresignExpiry, req.ContentType)
	if err != nil {
		response.Error(w, http.StatusBadGateway, "STORAGE_ERROR", "storage error")
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(PresignExpiry)

	up := &Upload{
		ID:           uploadID,
		UserID:       userID,
		Category:     Category("photo"), // если у вас есть константа CategoryPhoto — поставь её
		Status:       StatusStaged,      // если в entity это константа, иначе "staged"
		OriginalName: fileName,
		MimeType:     req.ContentType,
		Size:         req.FileSize,
		StagingKey:   stagingKey,
		CreatedAt:    now,
		ExpiresAt:    expiresAt,
	}

	if err := h.repo.Create(r.Context(), up); err != nil {
		response.Error(w, http.StatusInternalServerError, "DB_ERROR", "failed to save upload")
		return
	}

	response.OK(w, InitResponse{
		UploadID:  uploadID.String(),
		UploadURL: uploadURL,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}

// Confirm handles POST /uploads/confirm
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	var req ConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errs := validator.Validate(&req); errs != nil {
		response.ValidationError(w, errs)
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Unauthorized")
		return
	}

	uploadID, err := uuid.Parse(req.UploadID)
	if err != nil {
		response.BadRequest(w, "Invalid upload ID")
		return
	}

	up, err := h.repo.GetByID(r.Context(), uploadID)
	if err != nil {
		response.InternalError(w)
		return
	}
	if up == nil {
		response.NotFound(w, "Upload not found")
		return
	}
	if up.UserID != userID {
		response.Forbidden(w, "Not upload owner")
		return
	}

	exists, err := h.storage.Exists(r.Context(), up.StagingKey)
	if err != nil {
		response.Error(w, http.StatusBadGateway, "STORAGE_ERROR", "storage error")
		return
	}
	if !exists {
		response.Error(w, http.StatusBadRequest, "FILE_NOT_FOUND", "file not found in staging")
		return
	}

	now := time.Now().UTC()
	finalKey := fmt.Sprintf("uploads/final/%s/%d_%s", userID.String(), now.Unix(), sanitizeFileName(up.OriginalName))

	if err := h.storage.Move(r.Context(), up.StagingKey, finalKey); err != nil {
		response.Error(w, http.StatusBadGateway, "STORAGE_ERROR", "storage error")
		return
	}

	up.Status = StatusCommitted // или "committed"
	up.PermanentKey = finalKey
	up.PermanentURL = h.storage.GetURL(finalKey)
	up.CommittedAt = &now

	if err := h.repo.Update(r.Context(), up); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, ConfirmResponse{
		FileURL:   up.PermanentURL,
		FinalPath: finalKey,
	})
}

// Stage handles POST /uploads/stage
// Multipart form: file + category
func (h *Handler) Stage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)

	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		response.BadRequest(w, "File too large or invalid form")
		return
	}

	category := r.FormValue("category")
	if category == "" {
		category = "photo"
	}
	if category != "avatar" && category != "photo" && category != "document" {
		response.BadRequest(w, "Invalid category. Must be: avatar, photo, or document")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "No file provided")
		return
	}
	defer file.Close()

	userID := middleware.GetUserID(r.Context())

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

func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = path.Base(name)
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" {
		return "file"
	}
	return name
}
