package promotion

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// CastingPromotionHandler handles HTTP endpoints for casting promotions
type CastingPromotionHandler struct {
	repo          *CastingRepository
	creditService credit.Service
	validator     *validator.Validate
}

// NewCastingPromotionHandler creates a new CastingPromotionHandler
func NewCastingPromotionHandler(repo *CastingRepository, creditService credit.Service) *CastingPromotionHandler {
	return &CastingPromotionHandler{
		repo:          repo,
		creditService: creditService,
		validator:     validator.New(),
	}
}

// List returns all casting promotions for the current employer
// @Summary Список продвижений кастингов работодателя
// @Tags CastingPromotion
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401,404,500 {object} response.Response
// @Router /casting-promotions [get]
func (h *CastingPromotionHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	employerID, err := h.repo.GetEmployerIDByUserID(r.Context(), userID)
	if err == ErrProfileNotFound {
		response.NotFound(w, "employer profile not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	promotions, err := h.repo.GetByEmployerID(r.Context(), employerID)
	if err != nil {
		response.InternalError(w)
		return
	}

	var items []*CastingPromotionResponse
	for i := range promotions {
		items = append(items, promotions[i].ToCastingPromotionResponse())
	}

	response.OK(w, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// Create creates a new casting promotion (draft)
// @Summary Создать продвижение кастинга
// @Tags CastingPromotion
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateCastingPromotionRequest true "Данные продвижения"
// @Success 201 {object} response.Response{data=CastingPromotionResponse}
// @Failure 400,401,403,422,500 {object} response.Response
// @Router /casting-promotions [post]
func (h *CastingPromotionHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	employerID, err := h.repo.GetEmployerIDByUserID(r.Context(), userID)
	if err == ErrProfileNotFound {
		response.NotFound(w, "employer profile not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	var req CreateCastingPromotionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		response.BadRequest(w, err.Error())
		return
	}

	castingID, err := uuid.Parse(req.CastingID)
	if err != nil {
		response.BadRequest(w, "invalid casting_id")
		return
	}

	// Ensure employer owns this casting
	if err := h.repo.VerifyCastingOwner(r.Context(), castingID, employerID); err != nil {
		if err == ErrNotPromotionOwner {
			response.Forbidden(w, "you can only promote your own castings")
			return
		}
		response.InternalError(w)
		return
	}

	now := time.Now()
	cp := &CastingPromotion{
		ID:           uuid.New(),
		CastingID:    castingID,
		EmployerID:   employerID,
		BudgetAmount: req.BudgetAmount,
		DurationDays: req.DurationDays,
		Status:       StatusDraft,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.CustomTitle != "" {
		cp.CustomTitle = sql.NullString{String: req.CustomTitle, Valid: true}
	}
	if req.DailyBudget != nil {
		cp.DailyBudget = sql.NullInt64{Int64: *req.DailyBudget, Valid: true}
	}

	if err := h.repo.Create(r.Context(), cp); err != nil {
		response.InternalError(w)
		return
	}

	response.Created(w, cp.ToCastingPromotionResponse())
}

// Get returns a specific casting promotion by ID
// @Summary Получить продвижение кастинга
// @Tags CastingPromotion
// @Produce json
// @Param id path string true "ID продвижения"
// @Success 200 {object} response.Response{data=CastingPromotionResponse}
// @Failure 400,404,500 {object} response.Response
// @Router /casting-promotions/{id} [get]
func (h *CastingPromotionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid promotion id")
		return
	}

	cp, err := h.repo.GetByID(r.Context(), id)
	if err == ErrPromotionNotFound {
		response.NotFound(w, "promotion not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, cp.ToCastingPromotionResponse())
}

// Activate activates a casting promotion by deducting credits and starting it
// @Summary Активировать продвижение кастинга (списывает кредиты)
// @Tags CastingPromotion
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID продвижения"
// @Success 200 {object} response.Response{data=CastingPromotionResponse}
// @Failure 400,401,402,404,500 {object} response.Response
// @Router /casting-promotions/{id}/activate [post]
func (h *CastingPromotionHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid promotion id")
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "unauthorized")
		return
	}

	cp, err := h.repo.GetByID(r.Context(), id)
	if err == ErrPromotionNotFound {
		response.NotFound(w, "promotion not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	if !cp.CanBeActivated() {
		response.BadRequest(w, "promotion cannot be activated in its current status")
		return
	}

	// Deduct credits from user balance
	cost := int(cp.BudgetAmount)
	if err := h.creditService.Deduct(r.Context(), userID, cost, credit.TransactionMeta{
		Description:       fmt.Sprintf("Продвижение кастинга (ID: %s, %d дней)", cp.CastingID, cp.DurationDays),
		RelatedEntityType: "casting_promotion",
		RelatedEntityID:   cp.ID,
	}); err != nil {
		response.Error(w, http.StatusPaymentRequired, "INSUFFICIENT_CREDITS",
			"Недостаточно кредитов для активации продвижения")
		return
	}

	now := time.Now()
	startsAt := sql.NullTime{Time: now, Valid: true}
	endsAt := sql.NullTime{Time: now.AddDate(0, 0, cp.DurationDays), Valid: true}

	if err := h.repo.Activate(r.Context(), id, startsAt, endsAt, nil); err != nil {
		response.InternalError(w)
		return
	}

	cp, _ = h.repo.GetByID(r.Context(), id)
	response.OK(w, cp.ToCastingPromotionResponse())
}

// Pause pauses an active casting promotion
// @Summary Приостановить продвижение кастинга
// @Tags CastingPromotion
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID продвижения"
// @Success 200 {object} response.Response
// @Failure 400,401,404,500 {object} response.Response
// @Router /casting-promotions/{id}/pause [post]
func (h *CastingPromotionHandler) Pause(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid promotion id")
		return
	}

	if err := h.repo.UpdateStatus(r.Context(), id, StatusPaused); err != nil {
		if err == ErrPromotionNotFound {
			response.NotFound(w, "promotion not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": "paused"})
}

// GetStats returns analytics for a casting promotion
// @Summary Статистика продвижения кастинга
// @Tags CastingPromotion
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID продвижения"
// @Success 200 {object} response.Response
// @Failure 400,401,404,500 {object} response.Response
// @Router /casting-promotions/{id}/stats [get]
func (h *CastingPromotionHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid promotion id")
		return
	}

	cp, err := h.repo.GetByID(r.Context(), id)
	if err == ErrPromotionNotFound {
		response.NotFound(w, "promotion not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}

	dailyStats, _ := h.repo.GetDailyStats(r.Context(), id)

	var dailyItems []CastingDailyStatItem
	for _, s := range dailyStats {
		dailyItems = append(dailyItems, CastingDailyStatItem{
			Date:        s.Date.Format("2006-01-02"),
			Impressions: s.Impressions,
			Clicks:      s.Clicks,
			Responses:   s.Responses,
			Spent:       s.Spent,
		})
	}

	stats := &CastingPromotionStatsResponse{
		TotalImpressions: cp.Impressions,
		TotalClicks:      cp.Clicks,
		TotalResponses:   cp.Responses,
		TotalSpent:       cp.SpentAmount,
		DailyStats:       dailyItems,
	}
	if cp.Impressions > 0 {
		stats.CTR = float64(cp.Clicks) / float64(cp.Impressions) * 100
	}
	if cp.Clicks > 0 {
		stats.ConversionRate = float64(cp.Responses) / float64(cp.Clicks) * 100
	}

	response.OK(w, stats)
}

// CastingPromotionRoutes builds the chi router for casting promotions
func CastingPromotionRoutes(h *CastingPromotionHandler, authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Get("/{id}", h.Get)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", h.List)
		r.Post("/", h.Create)
		r.Post("/{id}/activate", h.Activate)
		r.Post("/{id}/pause", h.Pause)
		r.Get("/{id}/stats", h.GetStats)
	})
	// Make the r.Context() usable in functions (suppress unused warning)
	_ = context.Background()
	return r
}
