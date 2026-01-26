package subscription

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Service handles subscription business logic
type Service struct {
	repo Repository
}

// NewService creates subscription service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetPlans returns all available plans
func (s *Service) GetPlans(ctx context.Context) ([]*Plan, error) {
	return s.repo.ListPlans(ctx)
}

// GetPlan returns a specific plan
func (s *Service) GetPlan(ctx context.Context, planID PlanID) (*Plan, error) {
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil || plan == nil {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

// GetCurrentSubscription returns user's active subscription (or default Free)
func (s *Service) GetCurrentSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, *Plan, error) {
	sub, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	// If no subscription, return virtual "free" subscription
	if sub == nil {
		freePlan, _ := s.repo.GetPlanByID(ctx, PlanFree)
		return &Subscription{
			UserID:        userID,
			PlanID:        PlanFree,
			Status:        StatusActive,
			BillingPeriod: BillingMonthly,
			StartedAt:     time.Now(),
		}, freePlan, nil
	}

	plan, _ := s.repo.GetPlanByID(ctx, sub.PlanID)
	return sub, plan, nil
}

// Subscribe creates or upgrades a subscription
func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, req *SubscribeRequest) (*Subscription, error) {
	// Get target plan
	planID := PlanID(req.PlanID)
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil || plan == nil {
		return nil, ErrPlanNotFound
	}

	// Check if already subscribed to same plan
	existing, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if existing != nil && existing.PlanID == planID {
		return nil, ErrAlreadySubscribed
	}

	// Calculate expiry
	period := BillingPeriod(req.BillingPeriod)
	var expiresAt time.Time
	switch period {
	case BillingMonthly:
		expiresAt = time.Now().AddDate(0, 1, 0)
	case BillingYearly:
		expiresAt = time.Now().AddDate(1, 0, 0)
	default:
		return nil, ErrInvalidBillingPeriod
	}

	// Cancel existing subscription if any
	if existing != nil {
		_ = s.repo.Cancel(ctx, existing.ID, "Upgraded to "+string(planID))
	}

	// Create new subscription (status = pending until payment confirmed)
	now := time.Now()
	sub := &Subscription{
		ID:            uuid.New(),
		UserID:        userID,
		PlanID:        planID,
		StartedAt:     now,
		ExpiresAt:     sql.NullTime{Time: expiresAt, Valid: true},
		Status:        StatusPending, // Will be activated after payment
		BillingPeriod: period,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

// ActivateSubscription activates a pending subscription (after payment)
func (s *Service) ActivateSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	sub, err := s.repo.GetByID(ctx, subscriptionID)
	if err != nil || sub == nil {
		return ErrSubscriptionNotFound
	}

	sub.Status = StatusActive
	return s.repo.Update(ctx, sub)
}

// Cancel cancels an active subscription
func (s *Service) Cancel(ctx context.Context, userID uuid.UUID, reason string) error {
	sub, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil || sub == nil {
		return ErrSubscriptionNotFound
	}

	if sub.PlanID == PlanFree {
		return ErrCannotCancelFree
	}

	return s.repo.Cancel(ctx, sub.ID, reason)
}

// GetLimits returns user's current limits based on subscription
func (s *Service) GetLimits(ctx context.Context, userID uuid.UUID) (*Plan, error) {
	sub, plan, err := s.GetCurrentSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	// If subscription expired, return free limits
	if sub.IsExpired() {
		return s.repo.GetPlanByID(ctx, PlanFree)
	}

	return plan, nil
}

// CheckLimit checks if user is within their plan limits
func (s *Service) CheckLimit(ctx context.Context, userID uuid.UUID, limitType string, currentUsage int) (bool, *Plan, error) {
	plan, err := s.GetLimits(ctx, userID)
	if err != nil {
		return false, nil, err
	}

	switch limitType {
	case "photos":
		if currentUsage >= plan.MaxPhotos {
			return false, plan, nil
		}
	case "responses":
		if plan.MaxResponsesMonth != -1 && currentUsage >= plan.MaxResponsesMonth {
			return false, plan, nil
		}
	case "chat":
		if !plan.CanChat {
			return false, plan, nil
		}
	}

	return true, plan, nil
}

// ExpireOldSubscriptions expires subscriptions past their expiry date
func (s *Service) ExpireOldSubscriptions(ctx context.Context) (int, error) {
	return s.repo.ExpireOldSubscriptions(ctx)
}
