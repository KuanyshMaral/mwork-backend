package subscription

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// LimitsMiddleware provides middleware to enforce subscription limits.
type LimitsMiddleware struct {
	checker *LimitChecker
	service *Service
}

// NewLimitsMiddleware creates a middleware helper for subscription limits.
func NewLimitsMiddleware(service *Service) *LimitsMiddleware {
	return &LimitsMiddleware{
		checker: NewLimitChecker(service),
		service: service,
	}
}

// RequireResponseLimit enforces monthly response limits.
func (s *LimitsMiddleware) RequireResponseLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r.Context())
		if userID == uuid.Nil {
			response.Unauthorized(w, "unauthorized")
			return
		}

		usage, err := s.service.GetLimitsWithUsage(r.Context(), userID)
		if err != nil {
			response.InternalError(w)
			return
		}

		if err := s.checker.CanApplyToResponse(r.Context(), userID, usage.ResponsesUsed); err != nil {
			WriteLimitExceeded(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireChatAccess enforces chat availability by plan.
func (s *LimitsMiddleware) RequireChatAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r.Context())
		if userID == uuid.Nil {
			response.Unauthorized(w, "unauthorized")
			return
		}

		if err := s.checker.CanUseChat(r.Context(), userID); err != nil {
			WriteLimitExceeded(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequirePhotoUpload enforces photo upload limits.
func (s *LimitsMiddleware) RequirePhotoUpload(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r.Context())
		if userID == uuid.Nil {
			response.Unauthorized(w, "unauthorized")
			return
		}

		usage, err := s.service.GetLimitsWithUsage(r.Context(), userID)
		if err != nil {
			response.InternalError(w)
			return
		}

		if err := s.checker.CanUploadPhoto(r.Context(), userID, usage.PhotosUsed); err != nil {
			WriteLimitExceeded(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// WriteLimitExceeded renders a unified subscription limit error payload.
func WriteLimitExceeded(w http.ResponseWriter, err error) {
	var limitErr *LimitError
	if errors.As(err, &limitErr) {
		response.ErrorWithDetails(w, http.StatusTooManyRequests, "LIMIT_EXCEEDED", limitErr.Err.Error(), map[string]string{
			"current":    strconv.Itoa(limitErr.Current),
			"limit":      strconv.Itoa(limitErr.Limit),
			"plan_name":  limitErr.PlanName,
			"upgrade_to": limitErr.UpgradeTo,
		})
		return
	}
	response.InternalError(w)
}
