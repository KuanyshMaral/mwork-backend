package subscription

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Service handles subscription business logic
type Service struct {
	repo         Repository
	photoRepo    PhotoRepository
	responseRepo ResponseRepository
	castingRepo  CastingRepository
	profileRepo  ProfileRepository
}

// PhotoRepository interface for photo count operations
type PhotoRepository interface {
	CountByProfileID(ctx context.Context, profileID uuid.UUID) (int, error)
}

// ResponseRepository interface for response count operations
type ResponseRepository interface {
	CountWeeklyByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}

// CastingRepository interface for casting count operations
type CastingRepository interface {
	CountActiveByCreatorID(ctx context.Context, creatorID uuid.UUID) (int, error)
}

// ProfileRepository interface for profile operations
type ProfileRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (ProfileInfo, error)
}

// ProfileInfo interface for profile data
type ProfileInfo interface {
	GetID() uuid.UUID
}

// NewService creates subscription service
func NewService(repo Repository, photoRepo PhotoRepository, responseRepo ResponseRepository, castingRepo CastingRepository, profileRepo ProfileRepository) *Service {
	return &Service{
		repo:         repo,
		photoRepo:    photoRepo,
		responseRepo: responseRepo,
		castingRepo:  castingRepo,
		profileRepo:  profileRepo,
	}
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

// CreatePendingSubscription creates a pending subscription (before payment)
func (s *Service) CreatePendingSubscription(ctx context.Context, userID uuid.UUID, planID PlanID, billingPeriod string) (*Subscription, error) {
	// Get target plan
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
	period := BillingPeriod(billingPeriod)
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
		Status:        StatusPending,
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

// LimitsResponse represents current usage vs plan limits
type LimitsResponse struct {
	PhotosUsed     int `json:"photos_used"`
	PhotosLimit    int `json:"photos_limit"`
	ResponsesUsed  int `json:"responses_used"`
	ResponsesLimit int `json:"responses_limit"`
	CastingsUsed   int `json:"castings_used"`
	CastingsLimit  int `json:"castings_limit"`
}

// GetUserLimits returns current usage and plan limits for user
func (s *Service) GetUserLimits(ctx context.Context, userID uuid.UUID) (*LimitsResponse, error) {
	// Get active subscription or use free tier
	sub, err := s.repo.GetActiveByUserID(ctx, userID)
	var plan *Plan

	if err == sql.ErrNoRows || sub == nil {
		// Free tier defaults
		freePlan, err := s.repo.GetPlanByID(ctx, PlanFree)
		if err != nil {
			// Hardcoded fallback if free plan not in DB
			plan = &Plan{
				MaxPhotos:         3,
				MaxResponsesMonth: 5,
			}
		} else {
			plan = freePlan
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	} else {
		plan, err = s.repo.GetPlanByID(ctx, sub.PlanID)
		if err != nil {
			return nil, fmt.Errorf("failed to get plan: %w", err)
		}
	}

	// Get user's profile
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		log.Warn().Err(err).Str("user_id", userID.String()).Msg("Profile not found, returning zero usage")
		// User has no profile yet - return zero usage
		castingsLimit := 3 // Default casting limit
		if plan.ID == PlanPro {
			castingsLimit = 10
		} else if plan.ID == PlanAgency {
			castingsLimit = -1 // Unlimited
		}

		return &LimitsResponse{
			PhotosUsed:     0,
			PhotosLimit:    plan.MaxPhotos,
			ResponsesUsed:  0,
			ResponsesLimit: plan.MaxResponsesMonth,
			CastingsUsed:   0,
			CastingsLimit:  castingsLimit,
		}, nil
	}

	profileID := profile.GetID()

	// Get photo usage
	photosUsed, err := s.photoRepo.CountByProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to count photos: %w", err)
	}

	// Get response usage (weekly)
	responsesUsed, err := s.responseRepo.CountWeeklyByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count responses: %w", err)
	}

	// Get casting usage
	castingsUsed, err := s.castingRepo.CountActiveByCreatorID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to count castings: %w", err)
	}

	// Determine casting limit based on plan
	castingsLimit := 3 // Free tier default
	if plan.ID == PlanPro {
		castingsLimit = 10
	} else if plan.ID == PlanAgency {
		castingsLimit = -1 // Unlimited
	}

	log.Info().
		Str("user_id", userID.String()).
		Int("photos_used", photosUsed).
		Int("responses_used", responsesUsed).
		Int("castings_used", castingsUsed).
		Msg("Fetched user limits")

	return &LimitsResponse{
		PhotosUsed:     photosUsed,
		PhotosLimit:    plan.MaxPhotos,
		ResponsesUsed:  responsesUsed,
		ResponsesLimit: plan.MaxResponsesMonth,
		CastingsUsed:   castingsUsed,
		CastingsLimit:  castingsLimit,
	}, nil
}
