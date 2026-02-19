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
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) func(chi.Router) {
	return func(r chi.Router) {
		// Public: read attachments for any entity (e.g. model portfolio gallery)
		r.Get("/", h.List)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/", h.Attach)
			r.Delete("/{id}", h.Delete)
			r.Patch("/reorder", h.Reorder)
		})
	}
}

// attachRequest is the body for POST /attachments.
type attachRequest struct {
	UploadID   string   `json:"upload_id"`
	TargetType string   `json:"target_type"`
	TargetID   string   `json:"target_id"`
	Metadata   Metadata `json:"metadata"`
}

// reorderRequest is the body for PATCH /attachments/reorder.
type reorderRequest struct {
	IDs []uuid.UUID `json:"ids"`
}

// Attach handles POST /attachments
// Links an already-uploaded file (from POST /files) to a business entity.
func (h *Handler) Attach(w http.ResponseWriter, r *http.Request) {
	callerID := middleware.GetUserID(r.Context())
	if callerID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	var req attachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	uploadID, err := uuid.Parse(req.UploadID)
	if err != nil {
		response.BadRequest(w, "Invalid upload_id")
		return
	}
	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		response.BadRequest(w, "Invalid target_id")
		return
	}

	result, err := h.service.Attach(
		r.Context(),
		uploadID,
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

// List handles GET /attachments?target_type=model_portfolio&target_id={uuid}
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

// Delete handles DELETE /attachments/{id}
// Removes the link between file and entity. The file itself is NOT deleted.
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

// Reorder handles PATCH /attachments/reorder
// Accepts an ordered list of attachment IDs; updates sort_order accordingly.
func (h *Handler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req reorderRequest
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
