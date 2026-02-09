package photostudio_booking

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles PhotoStudio booking HTTP requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new PhotoStudio booking handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateBooking handles POST /api/v1/photostudio/bookings
// Requires authentication - extracts UserID from context.
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	// Extract authenticated user ID from context
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "User not authenticated")
		return
	}

	// Parse request body
	var req CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Create booking via PhotoStudio service
	booking, err := h.service.CreateBooking(r.Context(), userID, req)
	if err != nil {
		// Check for specific error types
		if isValidationError(err) {
			response.BadRequest(w, err.Error())
			return
		}
		if isConflictError(err) {
			response.Conflict(w, err.Error())
			return
		}
		response.InternalError(w)
		fmt.Println(err.Error())
		return
	}

	response.Created(w, booking)
}

// GetStudios handles GET /api/v1/photostudio/studios
// Public endpoint - no authentication required.
func (h *Handler) GetStudios(w http.ResponseWriter, r *http.Request) {
	// Parse query params
	city := r.URL.Query().Get("city")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Get studios from PhotoStudio service
	studios, err := h.service.GetStudios(r.Context(), city, page, limit)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, studios)
}

// Helper functions to identify error types
func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	// PhotoStudio returns VALIDATION_ERROR code
	return containsString(err.Error(), "VALIDATION_ERROR") ||
		containsString(err.Error(), "end_time must be after start_time")
}

func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	// PhotoStudio returns OVERBOOKING or NOT_AVAILABLE codes
	return containsString(err.Error(), "OVERBOOKING") ||
		containsString(err.Error(), "NOT_AVAILABLE") ||
		containsString(err.Error(), "status=409")
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
