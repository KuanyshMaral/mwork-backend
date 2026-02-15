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
	UploadID   string `json:"upload_id"`
	UploadURL  string `json:"upload_url,omitempty"`
	UploadMode string `json:"upload_mode"`
	ExpiresAt  string `json:"expires_at"`
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
	proxyUpload    bool

	storage UploadStorage
	repo    Repository
}

// NewHandler creates upload handler
func NewHandler(service *Service, stagingBaseURL string, st UploadStorage, repo Repository, proxyUpload bool) *Handler {
	return &Handler{
		service:        service,
		stagingBaseURL: stagingBaseURL,
		proxyUpload:    proxyUpload,
		storage:        st,
		repo:           repo,
	}
}

// Init handles POST /files/init
// @Summary Инициализировать загрузку файла
// @Description Purpose enum: avatar, portfolio, casting_cover, chat_file (and legacy photo/document where applicable).
// @Tags Upload
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body InitRequestDoc true "Метаданные файла"
// @Success 200 {object} response.Response{data=InitResponseDoc}
// @Failure 400,401,413,422,502,500 {object} response.ErrorResponse
// @Router /files/init [post]
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

	uploadMode := "direct"
	uploadURL := ""
	if h.proxyUpload {
		uploadMode = "proxy"
	} else {
		var err error
		uploadURL, err = h.storage.GeneratePresignedPutURL(r.Context(), stagingKey, PresignExpiry, req.ContentType)
		if err != nil {
			response.Error(w, http.StatusBadGateway, "STORAGE_ERROR", "storage error")
			return
		}
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
		UploadID:   uploadID.String(),
		UploadURL:  uploadURL,
		UploadMode: uploadMode,
		ExpiresAt:  expiresAt.Format(time.RFC3339),
	})
}

// Confirm handles POST /files/confirm
// @Summary Подтвердить загрузку файла
// @Description Confirm is idempotent: committed uploads return 200 with current fields.
// @Description Possible error codes: UPLOAD_NOT_FOUND, UPLOAD_FORBIDDEN, UPLOAD_EXPIRED, UPLOAD_INVALID_STATUS, INVALID_CONTENT_TYPE, FILE_TOO_LARGE, STORAGE_ERROR
// @Tags Upload
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ConfirmRequestDoc true "ID загрузки"
// @Success 200 {object} response.Response{data=ConfirmResponseDoc}
// @Failure 400,401,403,404,422,502,500 {object} response.ErrorResponse
// @Router /files/confirm [post]
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

	up, err := h.service.Confirm(r.Context(), uploadID, userID)
	if err != nil {
		switch err {
		case ErrUploadNotFound:
			response.NotFound(w, "Upload not found")
		case ErrNotUploadOwner:
			response.Forbidden(w, "Not upload owner")
		case ErrInvalidStatus:
			response.Error(w, http.StatusBadRequest, "INVALID_UPLOAD_STATUS", "invalid upload status")
		case ErrUploadExpired:
			response.Error(w, http.StatusGone, "UPLOAD_EXPIRED", "upload has expired")
		case ErrMetadataMismatch:
			response.Error(w, http.StatusBadRequest, "UPLOAD_METADATA_MISMATCH", "uploaded file metadata mismatch")
		case ErrStagingFileNotFound:
			response.Error(w, http.StatusBadRequest, "FILE_NOT_FOUND", "file not found in staging")
		default:
			response.Error(w, http.StatusBadGateway, "STORAGE_ERROR", "storage error")
		}
		return
	}

	response.OK(w, ConfirmResponse{
		FileURL:   up.PermanentURL,
		FinalPath: up.PermanentKey,
	})
}

// Stage handles POST /files/stage
// Multipart form: file + category
// @Summary Загрузить файл во временное хранилище
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param category formData string false "Категория (avatar|photo|document)"
// @Param file formData file true "Файл"
// @Success 201 {object} response.Response{data=UploadResponse}
// @Failure 400,500 {object} response.Response
// @Router /files/stage [post]
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

	uploadIDRaw := strings.TrimSpace(r.FormValue("upload_id"))

	var upload *Upload
	if uploadIDRaw != "" {
		uploadID, parseErr := uuid.Parse(uploadIDRaw)
		if parseErr != nil {
			response.BadRequest(w, "Invalid upload ID")
			return
		}
		upload, err = h.service.StageExisting(r.Context(), uploadID, userID, Category(category), header.Filename, file)
	} else {
		upload, err = h.service.Stage(r.Context(), userID, Category(category), header.Filename, file)
	}

	if err != nil {
		switch err {
		case storage.ErrFileTooLarge:
			response.BadRequest(w, "File exceeds maximum size")
		case storage.ErrInvalidMimeType:
			response.BadRequest(w, "File type not allowed")
		case storage.ErrEmptyFile:
			response.BadRequest(w, "File is empty")
		case ErrUploadNotFound:
			response.NotFound(w, "Upload not found")
		case ErrNotUploadOwner:
			response.Forbidden(w, "Not upload owner")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, UploadResponseFromEntity(upload, h.stagingBaseURL))
}

// Commit handles POST /files/{id}/commit
// @Summary Закоммитить загруженный файл
// @Tags Upload
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID загрузки"
// @Success 200 {object} response.Response{data=UploadResponse}
// @Failure 400,403,404,500 {object} response.Response
// @Router /files/{id}/commit [post]
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

// GetByID handles GET /files/{id}
// @Summary Получить загрузку по ID
// @Tags Upload
// @Produce json
// @Param id path string true "ID загрузки"
// @Success 200 {object} response.Response{data=UploadResponse}
// @Failure 400,404 {object} response.Response
// @Router /files/{id} [get]
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

// Delete handles DELETE /files/{id}
// @Summary Удалить загрузку
// @Tags Upload
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID загрузки"
// @Success 204 {string} string "No Content"
// @Failure 400,403,404,500 {object} response.Response
// @Router /files/{id} [delete]
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

// ListMy handles GET /files
// @Summary Список моих загрузок
// @Tags Upload
// @Produce json
// @Security BearerAuth
// @Param category query string false "Категория"
// @Success 200 {object} response.Response{data=[]UploadResponse}
// @Failure 500 {object} response.Response
// @Router /files [get]
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
