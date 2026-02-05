package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// AuditService interface for audit logging operations
// ✅ FIXED: Interface definition added
type AuditService interface {
	LogAdminAction(ctx context.Context, log AuditLog) error
}

// GrantCreditsRequest represents the request to grant credits
type GrantCreditsRequest struct {
	Amount int    `json:"amount" validate:"required,min=1,max=1000000"`
	Reason string `json:"reason" validate:"required,min=3,max=500"`
}

// CreditHandler handles admin credit operations
type CreditHandler struct {
	creditService credit.Service
	auditService  AuditService
}

// NewCreditHandler creates a new credit handler
func NewCreditHandler(creditService credit.Service, auditService AuditService) *CreditHandler {
	return &CreditHandler{
		creditService: creditService,
		auditService:  auditService,
	}
}

// GrantCredits handles POST /admin/users/{id}/credits/grant
// B3: Admin credit grant endpoint with permission check and audit logging
func (h *CreditHandler) GrantCredits(w http.ResponseWriter, r *http.Request) {
	// Parse user ID from URL
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	// Parse request body
	var req GrantCreditsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Validate amount range (1 to 1,000,000)
	if req.Amount < 1 || req.Amount > 1000000 {
		response.BadRequest(w, "Amount must be between 1 and 1,000,000")
		return
	}

	// Get admin ID from context
	adminID := middleware.GetUserID(r.Context())

	// Create credit metadata
	creditMeta := credit.TransactionMeta{
		RelatedEntityType: "admin_grant",
		RelatedEntityID:   uuid.New(),
		Description:       fmt.Sprintf("Admin grant by %s: %s", adminID.String(), req.Reason),
		AdminID:           &adminID,
	}

	// Grant credits
	err = h.creditService.Add(r.Context(), userID, req.Amount, credit.TransactionTypeAdminGrant, creditMeta)
	if err != nil {
		if err == credit.ErrUserNotFound {
			response.NotFound(w, "User not found")
			return
		}
		if err == credit.ErrInvalidAmount {
			response.BadRequest(w, "Invalid credit amount")
			return
		}
		response.InternalError(w)
		return
	}

	// Log audit trail
	if h.auditService != nil {
		go func() {
			bgCtx := context.Background()

			// ✅ FIXED: Prepare details as JSON
			detailsJSON, _ := json.Marshal(map[string]interface{}{
				"amount": req.Amount,
				"reason": req.Reason,
			})

			// ✅ FIXED: Use proper UUID types
			_ = h.auditService.LogAdminAction(bgCtx, AuditLog{
				ID:         uuid.New(),
				AdminID:    uuid.NullUUID{UUID: adminID, Valid: true},
				Action:     "credit.grant",
				EntityType: "user",
				EntityID:   uuid.NullUUID{UUID: userID, Valid: true},
				Details:    detailsJSON,
			})
		}()
	}

	// Get updated balance
	balance, err := h.creditService.GetBalance(r.Context(), userID)
	if err != nil {
		// Even if we can't get balance, the grant was successful
		balance = 0
	}

	response.OK(w, map[string]interface{}{
		"success": true,
		"message": "Credits granted successfully",
		"data": map[string]interface{}{
			"user_id":        userID,
			"amount_granted": req.Amount,
			"new_balance":    balance,
			"reason":         req.Reason,
		},
	})
}

// GetUserCredits handles GET /admin/users/{id}/credits
func (h *CreditHandler) GetUserCredits(w http.ResponseWriter, r *http.Request) {
	// Parse user ID from URL
	userIDStr := chi.URLParam(r, "id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	// Get balance
	balance, err := h.creditService.GetBalance(r.Context(), userID)
	if err != nil {
		if err == credit.ErrUserNotFound {
			response.NotFound(w, "User not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]interface{}{
		"user_id": userID,
		"balance": balance,
	})
}
