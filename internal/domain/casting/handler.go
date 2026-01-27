package casting

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCastingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	casting, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		switch err {
		case ErrOnlyEmployersCanCreate:
			response.Forbidden(w, "Only employers can create castings")
		default:
			response.InternalError(w)
		}
		return
	}

	response.Created(w, CastingResponseFromEntity(casting))
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	casting, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "Casting not found")
		return
	}

	if casting.Status == StatusDraft {
		userID := middleware.GetUserID(r.Context())
		if casting.CreatorID != userID {
			response.NotFound(w, "Casting not found")
			return
		}
	}

	go h.service.IncrementViewCount(context.Background(), id)

	response.OK(w, CastingResponseFromEntity(casting))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	var req UpdateCastingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	casting, err := h.service.Update(r.Context(), id, userID, &req)
	if err != nil {
		switch err {
		case ErrCastingNotFound:
			response.NotFound(w, "Casting not found")
		case ErrNotCastingOwner:
			response.Forbidden(w, "You can only edit your own castings")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, CastingResponseFromEntity(casting))
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	userID := middleware.GetUserID(r.Context())
	casting, err := h.service.UpdateStatus(r.Context(), id, userID, Status(req.Status))
	if err != nil {
		switch err {
		case ErrCastingNotFound:
			response.NotFound(w, "Casting not found")
		case ErrNotCastingOwner:
			response.Forbidden(w, "You can only manage your own castings")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, CastingResponseFromEntity(casting))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.Delete(r.Context(), id, userID); err != nil {
		switch err {
		case ErrCastingNotFound:
			response.NotFound(w, "Casting not found")
		case ErrNotCastingOwner:
			response.Forbidden(w, "You can only delete your own castings")
		default:
			response.InternalError(w)
		}
		return
	}

	response.NoContent(w)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := &Filter{}
	query := r.URL.Query()

	if q := query.Get("q"); q != "" {
		filter.Query = &q
	}
	if city := query.Get("city"); city != "" {
		filter.City = &city
	}
	if payMin := query.Get("pay_min"); payMin != "" {
		if v, err := strconv.ParseFloat(payMin, 64); err == nil {
			filter.PayMin = &v
		}
	}
	if payMax := query.Get("pay_max"); payMax != "" {
		if v, err := strconv.ParseFloat(payMax, 64); err == nil {
			filter.PayMax = &v
		}
	}

	sortBy := SortByNewest
	if s := query.Get("sort"); s != "" {
		switch s {
		case "pay_desc":
			sortBy = SortByPayDesc
		case "popular":
			sortBy = SortByPopular
		}
	}

	page, limit := 1, 20
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

	pagination := &Pagination{Page: page, Limit: limit}
	castings, total, err := h.service.List(r.Context(), filter, sortBy, pagination)
	if err != nil {
		response.InternalError(w)
		return
	}

	items := make([]*CastingResponse, len(castings))
	for i, c := range castings {
		items[i] = CastingResponseFromEntity(c)
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
