package subscription

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/domain/casting"
	"github.com/mwork/mwork-api/internal/domain/photo"
	"github.com/mwork/mwork-api/internal/domain/response"
)

type Service struct {
	repo         Repository
	photoRepo    photo.Repository
	responseRepo response.Repository
	castingRepo  casting.Repository
}

func NewService(repo Repository) *Service {
	type LimitsResponse struct {
		PhotosUsed     int `json:"photos_used"`
		PhotosLimit    int `json:"photos_limit"`
		ResponsesUsed  int `json:"responses_used"`
		ResponsesLimit int `json:"responses_limit"`
		CastingsUsed   int `json:"castings_used"`
		CastingsLimit  int `json:"castings_limit"`
	}

	return &Service{repo: repo}
}

func (s *Service) GetPlans(ctx context.Context) ([]*Plan, error) {
	return s.repo.ListPlans(ctx)
}

// GetLimits returns the plan limits for a user
func (s *Service) GetLimits(ctx context.Context, userID uuid.UUID) (*Plan, error) {
	_, plan, err := s.GetCurrentSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *Service) GetPlan(ctx context.Context, planID PlanID) (*Plan, error) {
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil || plan == nil {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

func (s *Service) GetCurrentSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, *Plan, error) {
	sub, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	if sub == nil {
		free, _ := s.repo.GetPlanByID(ctx, PlanFree)
		return &Subscription{
			UserID:        userID,
			PlanID:        PlanFree,
			Status:        StatusActive,
			BillingPeriod: BillingMonthly,
			StartedAt:     time.Now(),
		}, free, nil
	}

	plan, _ := s.repo.GetPlanByID(ctx, sub.PlanID)
	return sub, plan, nil
}

// CheckLimit checks if user can perform an action based on their plan limits
func (s *Service) CheckLimit(ctx context.Context, userID uuid.UUID, limitType string, currentUsage int) (bool, *Plan, error) {
	_, plan, err := s.GetCurrentSubscription(ctx, userID)
	if err != nil {
		return false, nil, err
	}

	switch limitType {
	case "photos":
		return currentUsage < plan.MaxPhotos, plan, nil
	case "responses":
		if plan.MaxResponsesMonth == -1 {
			return true, plan, nil // Unlimited
		}
		return currentUsage < plan.MaxResponsesMonth, plan, nil
	case "chat":
		return plan.CanChat, plan, nil
	default:
		return false, plan, nil
	}
}

// ActivateSubscription activates a pending subscription after payment
func (s *Service) ActivateSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	return s.repo.ActivateSubscription(ctx, subscriptionID)
}

func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, req *SubscribeRequest) (*Subscription, error) {
	planID := PlanID(req.PlanID)
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil || plan == nil {
		return nil, ErrPlanNotFound
	}

	existing, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.PlanID == planID {
		return nil, ErrAlreadySubscribed
	}

	var expiresAt time.Time
	switch BillingPeriod(req.BillingPeriod) {
	case BillingMonthly:
		expiresAt = time.Now().AddDate(0, 1, 0)
	case BillingYearly:
		expiresAt = time.Now().AddDate(1, 0, 0)
	default:
		return nil, ErrInvalidBillingPeriod
	}

	if existing != nil {
		_ = s.repo.Cancel(ctx, existing.ID, "Upgrade")
	}

	now := time.Now()
	sub := &Subscription{
		ID:            uuid.New(),
		UserID:        userID,
		PlanID:        planID,
		Status:        StatusPending,
		BillingPeriod: BillingPeriod(req.BillingPeriod),
		StartedAt:     now,
		ExpiresAt:     sql.NullTime{Time: expiresAt, Valid: true},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

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
