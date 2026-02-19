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

type PhotoRepository interface {
	CountByProfileID(ctx context.Context, profileID uuid.UUID) (int, error)
}

type ResponseRepository interface {
	CountMonthlyByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}

type CastingRepository interface {
	CountActiveByCreatorID(ctx context.Context, creatorID uuid.UUID) (int, error)
}

type ProfileRepository interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Profile, error)
}

type Profile struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func NewService(repo Repository, photoRepo PhotoRepository, responseRepo ResponseRepository, castingRepo CastingRepository, profileRepo ProfileRepository) *Service {
	return &Service{repo: repo, photoRepo: photoRepo, responseRepo: responseRepo, castingRepo: castingRepo, profileRepo: profileRepo}
}

func defaultFreePlan(audience Audience) *Plan {
	if audience == AudienceEmployer {
		return &Plan{ID: PlanFreeEmployer, Name: "Free Employer", Description: "Базовый бесплатный план для работодателей", PriceMonthly: 0, MaxPhotos: 10, MaxResponsesMonth: 0, CanChat: true, Audience: AudienceEmployer, IsActive: true}
	}
	return &Plan{ID: PlanFreeModel, Name: "Free Model", Description: "Базовый бесплатный план", PriceMonthly: 0, MaxPhotos: 3, MaxResponsesMonth: 20, CanChat: true, Audience: AudienceModel, IsActive: true}
}

func (s *Service) freePlanIDForAudience(a Audience) PlanID {
	if a == AudienceEmployer {
		return PlanFreeEmployer
	}
	return PlanFreeModel
}

func (s *Service) getUserAudience(ctx context.Context, userID uuid.UUID) (Audience, error) {
	return s.repo.GetUserRole(ctx, userID)
}

func (s *Service) GetPlans(ctx context.Context) ([]*Plan, error) { return s.repo.ListPlans(ctx) }

func (s *Service) GetPlan(ctx context.Context, planID PlanID) (*Plan, error) {
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil || plan == nil {
		return nil, ErrPlanNotFound
	}
	return plan, nil
}

func (s *Service) GetCurrentSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, *Plan, error) {
	audience, err := s.getUserAudience(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	sub, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if sub == nil {
		freeID := s.freePlanIDForAudience(audience)
		freePlan, _ := s.repo.GetPlanByID(ctx, freeID)
		if freePlan == nil {
			freePlan = defaultFreePlan(audience)
		}
		return &Subscription{UserID: userID, PlanID: freeID, Status: StatusActive, BillingPeriod: BillingMonthly, StartedAt: time.Now()}, freePlan, nil
	}
	plan, _ := s.repo.GetPlanByID(ctx, sub.PlanID)
	if plan == nil {
		plan = defaultFreePlan(audience)
	}
	return sub, plan, nil
}

func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, req *SubscribeRequest) (*Subscription, error) {
	planID := PlanID(req.PlanID)
	plan, err := s.repo.GetPlanByID(ctx, planID)
	if err != nil || plan == nil {
		return nil, ErrPlanNotFound
	}
	audience, err := s.getUserAudience(ctx, userID)
	if err != nil {
		return nil, err
	}
	if plan.Audience != audience {
		return nil, ErrPlanAudienceMismatch
	}
	existing, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.PlanID == planID {
		return nil, ErrAlreadySubscribed
	}
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
	if existing != nil {
		_ = s.repo.Cancel(ctx, existing.ID, "Upgraded to "+string(planID))
	}
	now := time.Now()
	sub := &Subscription{ID: uuid.New(), UserID: userID, PlanID: planID, StartedAt: now, ExpiresAt: sql.NullTime{Time: expiresAt, Valid: true}, Status: StatusPending, BillingPeriod: period, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

func (s *Service) ActivateSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	sub, err := s.repo.GetByID(ctx, subscriptionID)
	if err != nil || sub == nil {
		return ErrSubscriptionNotFound
	}
	sub.Status = StatusActive
	return s.repo.Update(ctx, sub)
}

func (s *Service) Cancel(ctx context.Context, userID uuid.UUID, reason string) error {
	sub, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil || sub == nil {
		return ErrSubscriptionNotFound
	}
	if sub.PlanID == PlanFree || sub.PlanID == PlanFreeEmployer || sub.PlanID == PlanFreeModel {
		return ErrCannotCancelFree
	}
	return s.repo.Cancel(ctx, sub.ID, reason)
}

func (s *Service) GetPlanLimits(ctx context.Context, userID uuid.UUID) (*Plan, error) {
	sub, plan, err := s.GetCurrentSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub.IsExpired() {
		audience, roleErr := s.getUserAudience(ctx, userID)
		if roleErr != nil {
			return nil, roleErr
		}
		return s.repo.GetPlanByID(ctx, s.freePlanIDForAudience(audience))
	}
	return plan, nil
}

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

func (s *Service) GetLimitsWithUsage(ctx context.Context, userID uuid.UUID) (*LimitsResponseData, error) {
	sub, plan, err := s.GetCurrentSubscription(ctx, userID)
	_ = sub
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	responsesRemaining := plan.MaxResponsesMonth
	if responsesRemaining < 0 {
		responsesRemaining = -1
	}
	now := time.Now()
	responsesResetAt := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	resp := &LimitsResponseData{PlanID: string(plan.ID), PlanName: plan.Name, PhotosLimit: plan.MaxPhotos, ResponsesLimit: plan.MaxResponsesMonth, ResponsesRemaining: responsesRemaining, ResponsesResetAt: responsesResetAt, CastingsLimit: 3}
	if profile == nil || err == sql.ErrNoRows {
		return resp, nil
	}
	photosUsed, _ := s.photoRepo.CountByProfileID(ctx, profile.ID)
	responsesUsed, _ := s.responseRepo.CountMonthlyByUserID(ctx, userID)
	castingsUsed, _ := s.castingRepo.CountActiveByCreatorID(ctx, profile.ID)
	resp.PhotosUsed, resp.ResponsesUsed, resp.CastingsUsed = photosUsed, responsesUsed, castingsUsed
	if resp.ResponsesLimit >= 0 {
		remaining := resp.ResponsesLimit - resp.ResponsesUsed
		if remaining < 0 {
			remaining = 0
		}
		resp.ResponsesRemaining = remaining
	}
	log.Info().Str("user_id", userID.String()).Int("responses_used", resp.ResponsesUsed).Int("responses_limit", resp.ResponsesLimit).Msg("limits fetched")
	return resp, nil
}

func (s *Service) GetLimitStatus(ctx context.Context, userID uuid.UUID, limitKey string) (*LimitStatus, error) {
	if !IsAllowedLimitKey(limitKey) {
		return nil, ErrInvalidLimitKey
	}
	plan, err := s.GetPlanLimits(ctx, userID)
	if err != nil {
		return nil, err
	}
	used, err := s.responseRepo.CountMonthlyByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	override, err := s.repo.GetLimitOverrideTotal(ctx, userID, limitKey)
	if err != nil {
		return nil, err
	}
	base := plan.MaxResponsesMonth
	remaining := -1
	if base >= 0 {
		final := base + override
		if final < 0 {
			final = 0
		}
		remaining = final - used
		if remaining < 0 {
			remaining = 0
		}
	}
	now := time.Now()
	resets := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	return &LimitStatus{LimitKey: limitKey, Base: base, Override: override, Used: used, Remaining: remaining, Period: "month", ResetsAt: &resets}, nil
}

func (s *Service) AdjustLimit(ctx context.Context, actorID, userID uuid.UUID, limitKey string, delta int, reason string) (*LimitStatus, error) {
	if !IsAllowedLimitKey(limitKey) {
		return nil, ErrInvalidLimitKey
	}
	status, err := s.GetLimitStatus(ctx, userID, limitKey)
	if err != nil {
		return nil, err
	}
	if status.Base >= 0 && status.Base+status.Override+delta < 0 {
		return nil, ErrLimitWouldBeNegative
	}
	if err := s.repo.CreateLimitOverride(ctx, &LimitOverride{ID: uuid.New(), UserID: userID, LimitKey: limitKey, Delta: delta, Reason: reason, CreatedBy: actorID, CreatedAt: time.Now()}); err != nil {
		return nil, err
	}
	return s.GetLimitStatus(ctx, userID, limitKey)
}

func (s *Service) SetLimit(ctx context.Context, actorID, userID uuid.UUID, limitKey string, value int, reason string) (*LimitStatus, error) {
	status, err := s.GetLimitStatus(ctx, userID, limitKey)
	if err != nil {
		return nil, err
	}
	if status.Base >= 0 && value < 0 {
		return nil, ErrLimitWouldBeNegative
	}
	finalNow := status.Base + status.Override
	if status.Base < 0 {
		return nil, ErrLimitWouldBeNegative
	}
	return s.AdjustLimit(ctx, actorID, userID, limitKey, value-finalNow, reason)
}

func (s *Service) GetAllLimitStatuses(ctx context.Context, userID uuid.UUID) ([]*LimitStatus, error) {
	item, err := s.GetLimitStatus(ctx, userID, LimitKeyCastingResponses)
	if err != nil {
		return nil, err
	}
	return []*LimitStatus{item}, nil
}

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
		status, err := s.GetLimitStatus(ctx, userID, LimitKeyCastingResponses)
		if err != nil {
			return false, nil, err
		}
		if status.Base >= 0 {
			final := status.Base + status.Override
			if currentUsage >= final {
				return false, plan, nil
			}
		}
	case "chat":
		if !plan.CanChat {
			return false, plan, nil
		}
	}
	return true, plan, nil
}

func (s *Service) ExpireOldSubscriptions(ctx context.Context) (int, error) {
	return s.repo.ExpireOldSubscriptions(ctx)
}
