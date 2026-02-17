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

// PhotoRepository defines photo operations needed by subscription
type PhotoRepository interface {
	CountByProfileID(ctx context.Context, profileID uuid.UUID) (int, error)
}

// ResponseRepository defines response operations needed by subscription
type ResponseRepository interface {
	CountMonthlyByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}

// CastingRepository defines casting operations needed by subscription
type CastingRepository interface {
	CountActiveByCreatorID(ctx context.Context, creatorID uuid.UUID) (int, error)
}

// ProfileRepository defines profile operations needed by subscription
type ProfileRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error)
}

// Profile represents a user profile (simplified for interface)
type Profile struct {
	ID     uuid.UUID
	UserID uuid.UUID
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

func defaultFreePlan() *Plan {
	return &Plan{
		ID:                PlanFree,
		Name:              "Free",
		Description:       "Базовый бесплатный план",
		PriceMonthly:      0,
		MaxPhotos:         3,
		MaxResponsesMonth: 20,
		CanChat:           true,
		CanSeeViewers:     false,
		PrioritySearch:    false,
		MaxTeamMembers:    0,
		IsActive:          true,
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
		if freePlan == nil {
			freePlan = defaultFreePlan()
		}
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

// GetPlanLimits returns user's current plan limits (renamed from GetLimits)
func (s *Service) GetPlanLimits(ctx context.Context, userID uuid.UUID) (*Plan, error) {
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

// LimitsResponseData represents current usage vs plan limits
type LimitsResponseData struct {
	PlanID             string    `json:"plan_id"`
	PlanName           string    `json:"plan_name"`
	PhotosUsed         int       `json:"photos_used"`
	PhotosLimit        int       `json:"photos_limit"`
	ResponsesUsed      int       `json:"responses_used"`
	ResponsesLimit     int       `json:"responses_limit"`
	ResponsesRemaining int       `json:"responses_remaining"`
	ResponsesResetAt   time.Time `json:"responses_reset_at"`
	CastingsUsed       int       `json:"castings_used"`
	CastingsLimit      int       `json:"castings_limit"`
}

// GetLimitsWithUsage returns current usage and plan limits for user
func (s *Service) GetLimitsWithUsage(ctx context.Context, userID uuid.UUID) (*LimitsResponseData, error) {
	// Get user subscription
	sub, err := s.repo.GetActiveByUserID(ctx, userID)
	var plan *Plan

	if err == sql.ErrNoRows || sub == nil {
		// Free tier defaults
		plan, err = s.repo.GetPlanByID(ctx, PlanFree)
		if err != nil {
			return nil, fmt.Errorf("failed to get free plan: %w", err)
		}
		if plan == nil {
			plan = defaultFreePlan()
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	} else {
		// Get plan from subscription
		plan, err = s.repo.GetPlanByID(ctx, sub.PlanID)
		if err != nil {
			return nil, fmt.Errorf("failed to get plan: %w", err)
		}
		if plan == nil {
			plan = defaultFreePlan()
		}
	}

	// Get user's profile for counting resources
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	// Initialize response with plan limits
	responsesRemaining := plan.MaxResponsesMonth
	if responsesRemaining < 0 {
		responsesRemaining = -1
	}

	now := time.Now()
	responsesResetAt := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())

	response := &LimitsResponseData{
		PlanID:             string(plan.ID),
		PlanName:           plan.Name,
		PhotosLimit:        plan.MaxPhotos,
		ResponsesLimit:     plan.MaxResponsesMonth,
		ResponsesRemaining: responsesRemaining,
		ResponsesResetAt:   responsesResetAt,
		CastingsLimit:      3, // Default for free plan
	}

	// If no profile, return zeroed usage
	if profile == nil || err == sql.ErrNoRows {
		response.PhotosUsed = 0
		response.ResponsesUsed = 0
		response.CastingsUsed = 0
		return response, nil
	}

	// Get actual usage counts in parallel for performance
	type usageResult struct {
		photos    int
		responses int
		castings  int
		err       error
	}

	resultChan := make(chan usageResult, 1)

	go func() {
		var result usageResult

		// Count photos
		photosUsed, err := s.photoRepo.CountByProfileID(ctx, profile.ID)
		if err != nil {
			log.Warn().Err(err).Msg("failed to count photos")
			photosUsed = 0
		}
		result.photos = photosUsed

		// Count monthly responses
		responsesUsed, err := s.responseRepo.CountMonthlyByUserID(ctx, userID)
		if err != nil {
			log.Warn().Err(err).Msg("failed to count responses")
			responsesUsed = 0
		}
		result.responses = responsesUsed

		// Count active castings
		castingsUsed, err := s.castingRepo.CountActiveByCreatorID(ctx, profile.ID)
		if err != nil {
			log.Warn().Err(err).Msg("failed to count castings")
			castingsUsed = 0
		}
		result.castings = castingsUsed

		resultChan <- result
	}()

	// Wait for results
	result := <-resultChan

	response.PhotosUsed = result.photos
	response.ResponsesUsed = result.responses
	response.CastingsUsed = result.castings
	if response.ResponsesLimit >= 0 {
		remaining := response.ResponsesLimit - response.ResponsesUsed
		if remaining < 0 {
			remaining = 0
		}
		response.ResponsesRemaining = remaining
	}

	log.Info().
		Str("user_id", userID.String()).
		Int("photos_used", response.PhotosUsed).
		Int("photos_limit", response.PhotosLimit).
		Int("responses_used", response.ResponsesUsed).
		Int("responses_limit", response.ResponsesLimit).
		Int("castings_used", response.CastingsUsed).
		Int("castings_limit", response.CastingsLimit).
		Msg("limits fetched")

	return response, nil
}

// CheckLimit checks if user is within their plan limits
func (s *Service) CheckLimit(ctx context.Context, userID uuid.UUID, limitType string, currentUsage int) (bool, *Plan, error) {
	plan, err := s.GetPlanLimits(ctx, userID)
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
