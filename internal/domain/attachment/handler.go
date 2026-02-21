package attachment

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// Handler handles attachment HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates attachment handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes registers the following endpoints:
//
//	POST   /attachments              - link an uploaded file to a target entity
//	GET    /attachments?target_type=&target_id= - list attachments for a target
//	DELETE /attachments/{id}         - unlink an attachment (does NOT delete the file)
//	PATCH  /attachments/reorder      - reorder attachments within a target
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public: read attachments for any entity (e.g. model portfolio gallery)
	r.Get("/", h.List)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", h.Attach)
		r.Delete("/{id}", h.Delete)
		r.Patch("/reorder", h.Reorder)
	})

	return r
}

// AttachRequest is the body for POST /attachments.
// @Description Параметры для привязки файлов к сущности
type AttachRequest struct {
	UploadIDs  []string `json:"upload_ids" example:"[\"uuid1\", \"uuid2\"]"`
	TargetType string   `json:"target_type" example:"model_portfolio"`
	TargetID   string   `json:"target_id" example:"uuid"`
	Metadata   Metadata `json:"metadata"`
}

// ReorderRequest is the body for PATCH /attachments/reorder.
type ReorderRequest struct {
	IDs []uuid.UUID `json:"ids"`
}

// swaggerListAttachmentResponse is a wrapper strictly for generating Swagger documentation.
type swaggerListAttachmentResponse struct {
	Success bool                `json:"success"`
	Data    []AttachmentWithURL `json:"data"`
}

// @Summary Привязать один или несколько файлов к сущности
// @Description Создает связи между ранее загруженными файлами (через POST /files) и бизнес-сущностью (например, портфолио модели).
// @Tags Entity Attachments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body AttachRequest true "Данные для привязки"
// @Success 201 {object} swaggerListAttachmentResponse "Успешная привязка"
// @Failure 400 {object} response.ErrorResponse "Неверные данные"
// @Failure 401 {object} response.ErrorResponse "Не авторизован"
// @Failure 403 {object} response.ErrorResponse "Нет прав на использование одного из файлов"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /attachments [post]
func (h *Handler) Attach(w http.ResponseWriter, r *http.Request) {
	callerID := middleware.GetUserID(r.Context())
	if callerID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	var req AttachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if len(req.UploadIDs) == 0 {
		response.BadRequest(w, "At least one upload_id is required")
		return
	}

	uploadIDs := make([]uuid.UUID, len(req.UploadIDs))
	for i, idStr := range req.UploadIDs {
		uid, err := uuid.Parse(idStr)
		if err != nil {
			response.BadRequest(w, "Invalid upload_id: "+idStr)
			return
		}
		uploadIDs[i] = uid
	}

	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		response.BadRequest(w, "Invalid target_id")
		return
	}

	result, err := h.service.Attach(
		r.Context(),
		uploadIDs,
		callerID,
		TargetType(req.TargetType),
		targetID,
		req.Metadata,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotOwner):
			response.Forbidden(w, "You do not own this file")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, result)
}

// @Summary Получить список вложений сущности
// @Tags Entity Attachments
// @Produce json
// @Param target_type query string true "Тип сущности (например, model_portfolio)"
// @Param target_id query string true "ID сущности (UUID)"
// @Success 200 {object} swaggerListAttachmentResponse "Список вложений"
// @Failure 400 {object} response.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /attachments [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	targetTypeStr := r.URL.Query().Get("target_type")
	targetIDStr := r.URL.Query().Get("target_id")

	if targetTypeStr == "" || targetIDStr == "" {
		response.BadRequest(w, "target_type and target_id query params are required")
		return
	}

	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		response.BadRequest(w, "Invalid target_id")
		return
	}

	items, err := h.service.ListByTarget(r.Context(), TargetType(targetTypeStr), targetID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, items)
}

// @Summary Удалить вложение
// @Description Удаляет связь между файлом и сущностью. Сам файл НЕ удаляется.
// @Tags Entity Attachments
// @Security BearerAuth
// @Param id path string true "ID вложения (UUID)"
// @Success 204 {string} string "Успешное удаление"
// @Failure 400 {object} response.ErrorResponse "Неверный ID"
// @Failure 401 {object} response.ErrorResponse "Не авторизован"
// @Failure 403 {object} response.ErrorResponse "Нет прав на удаление"
// @Failure 404 {object} response.ErrorResponse "Вложение не найден"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /attachments/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	callerID := middleware.GetUserID(r.Context())
	if callerID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid attachment ID")
		return
	}

	if err := h.service.Delete(r.Context(), id, callerID); err != nil {
		switch {
		case errors.Is(err, ErrAttachmentNotFound):
			response.NotFound(w, "Attachment not found")
		case errors.Is(err, ErrNotOwner):
			response.Forbidden(w, "You do not own this attachment")
		default:
			response.InternalError(w)
		}
		return
	}

	response.NoContent(w)
}

// swaggerReorderResponse is a wrapper strictly for generating Swagger documentation.
type swaggerReorderResponse struct {
	Success bool              `json:"success"`
	Data    map[string]string `json:"data"`
}

// @Summary Изменить порядок вложений
// @Description Принимает упорядоченный список ID вложений и обновляет их sort_order.
// @Tags Entity Attachments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body ReorderRequest true "Массив ID вложений в нужном порядке"
// @Success 200 {object} swaggerReorderResponse "Успешное обновление"
// @Failure 400 {object} response.ErrorResponse "Неверные данные"
// @Failure 500 {object} response.ErrorResponse "Внутренняя ошибка сервера"
// @Router /attachments/reorder [patch]
func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req ReorderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}
	if len(req.IDs) == 0 {
		response.BadRequest(w, "ids array must not be empty")
		return
	}

	if err := h.service.Reorder(r.Context(), req.IDs); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "ok"})
}
