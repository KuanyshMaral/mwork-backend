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

// UploadResponse is the API response shape for an upload.
// @Description Данные о загруженном файле
type UploadResponse struct {
	ID           uuid.UUID `json:"id"`
	URL          string    `json:"url"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    string    `json:"created_at"`
}

func (h *Handler) toResponse(u *Upload) *UploadResponse {
	return &UploadResponse{
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
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Post("/", h.Upload)       // POST /files
	r.Get("/{id}", h.Get)       // GET  /files/{id}
	r.Delete("/{id}", h.Delete) // DELETE /files/{id}

	return r
}

// swaggerListUploadResponse is a wrapper strictly for generating Swagger documentation.
type swaggerListUploadResponse struct {
	Success bool             `json:"success"`
	Data    []UploadResponse `json:"data"`
}

// @Summary Загрузка одного или нескольких файлов
// @Description Принимает multipart/form-data с одним или несколькими полями "file". Возвращает список метаданных загруженных файлов.
// @Tags Uploads
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Файл для загрузки (можно передать несколько полей 'file')"
// @Success 201 {object} swaggerListUploadResponse "Успешная загрузка"
// @Failure 400 {object} response.ErrorResponse "Ошибка запроса"
// @Failure 401 {object} response.ErrorResponse "Не авторизован"
// @Failure 413 {object} response.ErrorResponse "Файл слишком большой"
// @Failure 422 {object} response.ErrorResponse "Неподдерживаемый тип файла"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /files [post]
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

	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		response.BadRequest(w, "Missing 'file' field(s) in form data")
		return
	}

	results := make([]*UploadResponse, 0, len(files))
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			response.InternalError(w)
			return
		}

		upload, err := h.service.Upload(r.Context(), authorID, header.Filename, file)
		file.Close() // Close immediately after reading

		if err != nil {
			switch {
			case errors.Is(err, ErrFileTooLarge):
				response.Error(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "File "+header.Filename+" exceeds the maximum allowed size (50 MB)")
			case errors.Is(err, ErrInvalidMime):
				response.Error(w, http.StatusUnprocessableEntity, "INVALID_FILE_TYPE", "File type of "+header.Filename+" is not allowed")
			default:
				response.InternalError(w)
			}
			return // For now, we abort on first error to keep it simple and consistent
		}
		results = append(results, h.toResponse(upload))
	}

	response.Created(w, results)
}

// swaggerUploadResponse is a wrapper strictly for generating Swagger documentation.
type swaggerUploadResponse struct {
	Success bool           `json:"success"`
	Data    UploadResponse `json:"data"`
}

// @Summary Получить метаданные файла
// @Description Возвращает информацию о файле по его ID. Не возвращает сам файл — используйте поле 'url' для доступа к файлу.
// @Tags Uploads
// @Produce json
// @Param id path string true "ID файла (UUID)"
// @Success 200 {object} swaggerUploadResponse "Успешное получение"
// @Failure 400 {object} response.ErrorResponse "Неверный ID"
// @Failure 404 {object} response.ErrorResponse "Файл не найден"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /files/{id} [get]
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

// @Summary Удалить файл
// @Description Удаляет файл с диска и его метаданные из базы данных. Только для владельца файла.
// @Tags Uploads
// @Security BearerAuth
// @Param id path string true "ID файла (UUID)"
// @Success 204 {string} string "Успешное удаление"
// @Failure 400 {object} response.ErrorResponse "Неверный ID"
// @Failure 401 {object} response.ErrorResponse "Не авторизован"
// @Failure 403 {object} response.ErrorResponse "Нет прав на удаление"
// @Failure 404 {object} response.ErrorResponse "Файл не найден"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /files/{id} [delete]
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
