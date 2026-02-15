package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

type Handler struct {
	svc *Service
}

type walletRequest struct {
	Amount      int64  `json:"amount"`
	ReferenceID string `json:"reference_id"`
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) TopUp(w http.ResponseWriter, r *http.Request) {
	h.handleMutation(w, r, h.svc.TopUp)
}

func (h *Handler) Spend(w http.ResponseWriter, r *http.Request) {
	h.handleMutation(w, r, h.svc.Spend)
}

func (h *Handler) Refund(w http.ResponseWriter, r *http.Request) {
	h.handleMutation(w, r, h.svc.Refund)
}

func (h *Handler) Balance(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	balance, err := h.svc.GetBalance(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]interface{}{"balance": balance})
}

func (h *Handler) handleMutation(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	var req walletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	err := fn(r.Context(), userID, req.Amount, req.ReferenceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAmount):
			response.BadRequest(w, "amount must be greater than zero and reference_id is required for spend/refund")
		case errors.Is(err, ErrInsufficientFunds):
			response.Conflict(w, "insufficient wallet balance")
		case errors.Is(err, ErrReferenceConflict):
			response.Conflict(w, "reference_id already used with a different amount")
		default:
			response.InternalError(w)
		}
		return
	}

	balance, err := h.svc.GetBalance(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]interface{}{"balance": balance})
}

func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)
	r.Post("/topup", h.TopUp)
	r.Post("/spend", h.Spend)
	r.Post("/refund", h.Refund)
	r.Get("/balance", h.Balance)
	return r
}
