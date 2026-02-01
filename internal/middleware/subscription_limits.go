package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/pkg/response"
)

// SubscriptionLimitChecker provides limit checks for subscription-based actions.
type SubscriptionLimitChecker interface {
	CanApplyToResponse(ctx context.Context, userID uuid.UUID, monthlyApplications int) error
	CanUseChat(ctx context.Context, userID uuid.UUID) error
	CanUploadPhoto(ctx context.Context, userID uuid.UUID, currentPhotoCount int) error
}

// ResponseCounter provides monthly response counts per user.
type ResponseCounter interface {
	CountMonthlyByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}

// PhotoCounter provides photo counts per profile.
type PhotoCounter interface {
	CountByProfileID(ctx context.Context, profileID uuid.UUID) (int, error)
}

// ProfileIDProvider provides profile IDs by user.
type ProfileIDProvider interface {
	ProfileIDByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
}

// LimitErrorDetails describes the upgrade information for a limit error.
type LimitErrorDetails interface {
	error
	CurrentValue() int
	LimitValue() int
	PlanNameValue() string
	UpgradeToValue() string
}

// LimitPayload is the 429 response body for subscription limits.
type LimitPayload struct {
	Message   string `json:"message"`
	Current   int    `json:"current"`
	Limit     int    `json:"limit"`
	PlanName  string `json:"plan_name"`
	UpgradeTo string `json:"upgrade_to"`
	PlanID    string `json:"plan_id"`
}

// WriteLimitExceeded writes a structured 429 response if the error is a limit error.
func WriteLimitExceeded(w http.ResponseWriter, err error) bool {
	payload, ok := buildLimitPayload(err)
	if !ok {
		return false
	}

	response.ErrorWithDetails(
		w,
		http.StatusTooManyRequests,
		"LIMIT_EXCEEDED",
		payload.Message,
		buildLimitDetails(payload),
	)
	return true
}

func buildLimitPayload(err error) (*LimitPayload, bool) {
	var limitErr LimitErrorDetails
	if !errors.As(err, &limitErr) {
		return nil, false
	}

	return &LimitPayload{
		Message:   err.Error(),
		Current:   limitErr.CurrentValue(),
		Limit:     limitErr.LimitValue(),
		PlanName:  limitErr.PlanNameValue(),
		UpgradeTo: limitErr.UpgradeToValue(),
		PlanID:    limitErr.PlanNameValue(),
	}, true
}

func buildLimitDetails(payload *LimitPayload) map[string]string {
	details := map[string]string{
		"current":    strconv.Itoa(payload.Current),
		"limit":      strconv.Itoa(payload.Limit),
		"plan_name":  payload.PlanName,
		"upgrade_to": payload.UpgradeTo,
	}
	if payload.PlanID != "" {
		details["plan_id"] = payload.PlanID
	}
	return details
}

// RequireResponseLimit enforces monthly response limits.
func RequireResponseLimit(checker SubscriptionLimitChecker, counter ResponseCounter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if checker == nil || counter == nil {
				next.ServeHTTP(w, r)
				return
			}

			userID := GetUserID(r.Context())
			if userID == uuid.Nil {
				response.Unauthorized(w, "unauthorized")
				return
			}

			count, err := counter.CountMonthlyByUserID(r.Context(), userID)
			if err != nil {
				response.InternalError(w)
				return
			}

			if err := checker.CanApplyToResponse(r.Context(), userID, count); err != nil {
				if WriteLimitExceeded(w, err) {
					return
				}
				response.InternalError(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireChatLimit enforces chat access limits.
func RequireChatLimit(checker SubscriptionLimitChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if checker == nil {
				next.ServeHTTP(w, r)
				return
			}

			userID := GetUserID(r.Context())
			if userID == uuid.Nil {
				response.Unauthorized(w, "unauthorized")
				return
			}

			if err := checker.CanUseChat(r.Context(), userID); err != nil {
				if WriteLimitExceeded(w, err) {
					return
				}
				response.InternalError(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePhotoLimit enforces photo upload limits.
func RequirePhotoLimit(checker SubscriptionLimitChecker, counter PhotoCounter, profiles ProfileIDProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if checker == nil || counter == nil || profiles == nil {
				next.ServeHTTP(w, r)
				return
			}

			userID := GetUserID(r.Context())
			if userID == uuid.Nil {
				response.Unauthorized(w, "unauthorized")
				return
			}

			profileID, err := profiles.ProfileIDByUserID(r.Context(), userID)
			if err != nil {
				response.InternalError(w)
				return
			}
			if profileID == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}

			count, err := counter.CountByProfileID(r.Context(), profileID)
			if err != nil {
				response.InternalError(w)
				return
			}

			if err := checker.CanUploadPhoto(r.Context(), userID, count); err != nil {
				if WriteLimitExceeded(w, err) {
					return
				}
				response.InternalError(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
