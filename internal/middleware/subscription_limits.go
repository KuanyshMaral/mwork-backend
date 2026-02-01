package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/subscription"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// SubscriptionLimits provides middleware to enforce subscription limits.
type SubscriptionLimits struct {
	checker *subscription.LimitChecker
	service *subscription.Service
}

// NewSubscriptionLimits creates a middleware helper for subscription limits.
func NewSubscriptionLimits(service *subscription.Service) *SubscriptionLimits {
	return &SubscriptionLimits{
		checker: subscription.NewLimitChecker(service),
		service: service,
	}
}

// RequireResponseLimit enforces monthly response limits.
func (s *SubscriptionLimits) RequireResponseLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
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
func (s *SubscriptionLimits) RequireChatAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
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
func (s *SubscriptionLimits) RequirePhotoUpload(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := GetUserID(r.Context())
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
	var limitErr *subscription.LimitError
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
