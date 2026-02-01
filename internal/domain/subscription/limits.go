package subscription

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Limit errors
var (
	ErrPhotoLimitReached    = errors.New("photo upload limit reached for your plan")
	ErrResponseLimitReached = errors.New("monthly response limit reached for your plan")
	ErrChatNotAllowed       = errors.New("chat is not available on your current plan")
)

// LimitChecker provides convenience methods for checking subscription limits
type LimitChecker struct {
	svc *Service
}

// NewLimitChecker creates a limit checker
func NewLimitChecker(svc *Service) *LimitChecker {
	return &LimitChecker{svc: svc}
}

// CanUploadPhoto checks if user can upload more photos
func (c *LimitChecker) CanUploadPhoto(ctx context.Context, userID uuid.UUID, currentPhotoCount int) error {
	allowed, plan, err := c.svc.CheckLimit(ctx, userID, "photos", currentPhotoCount)
	if err != nil {
		return err
	}
	if !allowed {
		return &LimitError{
			Err:       ErrPhotoLimitReached,
			Current:   currentPhotoCount,
			Limit:     plan.MaxPhotos,
			PlanName:  string(plan.ID),
			UpgradeTo: c.getUpgradePlan(plan.ID),
		}
	}
	return nil
}

// CanApplyToResponse checks if user can apply to more castings this month
func (c *LimitChecker) CanApplyToResponse(ctx context.Context, userID uuid.UUID, monthlyApplications int) error {
	allowed, plan, err := c.svc.CheckLimit(ctx, userID, "responses", monthlyApplications)
	if err != nil {
		return err
	}
	if !allowed {
		return &LimitError{
			Err:       ErrResponseLimitReached,
			Current:   monthlyApplications,
			Limit:     plan.MaxResponsesMonth,
			PlanName:  string(plan.ID),
			UpgradeTo: c.getUpgradePlan(plan.ID),
		}
	}
	return nil
}

// CanUseChat checks if user can use chat feature
func (c *LimitChecker) CanUseChat(ctx context.Context, userID uuid.UUID) error {
	allowed, plan, err := c.svc.CheckLimit(ctx, userID, "chat", 0)
	if err != nil {
		return err
	}
	if !allowed {
		return &LimitError{
			Err:       ErrChatNotAllowed,
			PlanName:  string(plan.ID),
			UpgradeTo: c.getUpgradePlan(plan.ID),
		}
	}
	return nil
}

// GetLimitsStatus returns current limits status for UI display
func (c *LimitChecker) GetLimitsStatus(ctx context.Context, userID uuid.UUID) (*LimitsStatus, error) {
	_, plan, err := c.svc.GetCurrentSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &LimitsStatus{
		PlanID:            string(plan.ID),
		PlanName:          plan.Name,
		MaxPhotos:         plan.MaxPhotos,
		MaxResponsesMonth: plan.MaxResponsesMonth,
		CanChat:           plan.CanChat,
		CanSeeViewers:     plan.CanSeeViewers,
		PrioritySearch:    plan.PrioritySearch,
	}, nil
}

func (c *LimitChecker) getUpgradePlan(currentPlan PlanID) string {
	switch currentPlan {
	case PlanFree:
		return string(PlanPro)
	case PlanPro:
		return string(PlanAgency)
	default:
		return ""
	}
}

// LimitError contains detailed limit information for UI
type LimitError struct {
	Err       error
	Current   int
	Limit     int
	PlanName  string
	UpgradeTo string
}

func (e *LimitError) CurrentValue() int {
	return e.Current
}

func (e *LimitError) LimitValue() int {
	return e.Limit
}

func (e *LimitError) PlanNameValue() string {
	return e.PlanName
}

func (e *LimitError) UpgradeToValue() string {
	return e.UpgradeTo
}

func (e *LimitError) Error() string {
	return e.Err.Error()
}

func (e *LimitError) Unwrap() error {
	return e.Err
}

// LimitsStatus represents current limits for UI
type LimitsStatus struct {
	PlanID            string `json:"plan_id"`
	PlanName          string `json:"plan_name"`
	MaxPhotos         int    `json:"max_photos"`
	MaxResponsesMonth int    `json:"max_responses_month"` // -1 = unlimited
	CanChat           bool   `json:"can_chat"`
	CanSeeViewers     bool   `json:"can_see_viewers"`
	PrioritySearch    bool   `json:"priority_search"`
}
