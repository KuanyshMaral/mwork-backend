package response

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/domain/casting"
	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/domain/profile"
	"github.com/mwork/mwork-api/internal/pkg/featurepayment"
)

// NotificationService interface for notification operations
type NotificationService interface {
	NotifyNewResponse(ctx context.Context, employerUserID uuid.UUID, castingID uuid.UUID, responseID uuid.UUID, castingTitle string, modelName string) error
	NotifyResponseStatusChange(ctx context.Context, modelUserID uuid.UUID, castingTitle string, status string, castingID uuid.UUID, responseID uuid.UUID) error
}

// ChatServiceInterface interface for chat room operations
// This interface matches chat.Service to enable auto-creation of chat rooms on response acceptance
type ChatServiceInterface interface {
	CreateOrGetRoom(ctx context.Context, userID uuid.UUID, req *ChatRoomRequest) (*ChatRoom, error)
}

// ChatRoomRequest is a DTO for creating chat rooms (to avoid import cycle with chat package)
type ChatRoomRequest struct {
	RecipientID uuid.UUID
	CastingID   *uuid.UUID
	Message     string
}

// ChatRoom is a DTO for chat room response (to avoid import cycle with chat package)
type ChatRoom struct {
	ID             uuid.UUID
	Participant1ID uuid.UUID
	Participant2ID uuid.UUID
}

// Service handles response business logic
type Service struct {
	repo            Repository
	castingRepo     casting.Repository
	modelRepo       profile.ModelRepository
	employerRepo    profile.EmployerRepository
	notifService    NotificationService
	creditSvc       credit.Service
	paymentProvider featurepayment.PaymentProvider
	chatSvc         ChatServiceInterface
	limitChecker    SubLimitChecker
}

// NewService creates response service
func NewService(repo Repository, castingRepo casting.Repository, modelRepo profile.ModelRepository, employerRepo profile.EmployerRepository) *Service {
	return &Service{
		repo:         repo,
		castingRepo:  castingRepo,
		modelRepo:    modelRepo,
		employerRepo: employerRepo,
	}
}

// SetNotificationService sets the notification service (optional, to avoid circular dependency)
func (s *Service) SetNotificationService(notifService NotificationService) {
	s.notifService = notifService
}

// SetCreditService sets the credit service (optional, to avoid circular dependency)
func (s *Service) SetCreditService(creditSvc credit.Service) {
	s.creditSvc = creditSvc
}

// SetPaymentProvider sets unified payment provider for paid feature charges.
func (s *Service) SetPaymentProvider(paymentProvider featurepayment.PaymentProvider) {
	s.paymentProvider = paymentProvider
}

// SetChatService sets the chat service (optional, to avoid circular dependency)
func (s *Service) SetChatService(chatSvc ChatServiceInterface) {
	s.chatSvc = chatSvc
}

// Apply applies to a casting
// SubLimitChecker checks subscription-based response limits.
type SubLimitChecker interface {
	CanApplyToResponse(ctx context.Context, userID uuid.UUID, monthlyApplications int) error
}

// SetLimitChecker sets the subscription limit checker (optional, to avoid circular dependency).
func (s *Service) SetLimitChecker(lc SubLimitChecker) {
	s.limitChecker = lc
}

// Apply applies to a casting with hybrid billing:
//  1. Validate requirements (BEFORE billing)
//  2. Check monthly subscription limit
//  3. If within limit → create response (non-tx)
//  4. If limit exhausted → open tx, DeductTx (FOR UPDATE), CreateTx, Commit
//  5. Any failure → Rollback
func (s *Service) Apply(ctx context.Context, userID uuid.UUID, castingID uuid.UUID, req *ApplyRequest) (*Response, error) {
	// Get user's model profile
	prof, err := s.modelRepo.GetByUserID(ctx, userID)
	if err != nil || prof == nil {
		return nil, ErrProfileRequired
	}

	// Get casting
	cast, err := s.castingRepo.GetByID(ctx, castingID)
	if err != nil || cast == nil {
		return nil, ErrCastingNotFound
	}

	// Check if casting is active
	if !cast.IsActive() {
		return nil, ErrCastingNotActive
	}

	// Check if already applied
	existing, _ := s.repo.GetByModelAndCasting(ctx, prof.ID, castingID)
	if existing != nil {
		return nil, ErrAlreadyApplied
	}

	if isUrgentDifferentCity(cast, prof) {
		return nil, ErrGeoBlocked
	}

	// Phase 1: Check casting requirements BEFORE any billing
	if violations := CheckRequirements(cast, prof); len(violations) > 0 {
		return nil, &RequirementsError{Details: violations}
	}

	// Build response entity
	now := time.Now()
	resp := &Response{
		ID:        uuid.New(),
		CastingID: castingID,
		ModelID:   prof.ID,
		Status:    StatusPending,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if req.Message != "" {
		resp.Message = sql.NullString{String: req.Message, Valid: true}
	}
	if req.ProposedRate != nil {
		resp.ProposedRate = sql.NullFloat64{Float64: *req.ProposedRate, Valid: true}
	}

	// Phase 2: Hybrid billing — subscription first, then credits
	usedCredit := false
	withinLimit := true

	if s.limitChecker != nil {
		count, err := s.repo.CountMonthlyByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if err := s.limitChecker.CanApplyToResponse(ctx, userID, count); err != nil {
			// Subscription limit exhausted — will try credits
			withinLimit = false
		}
	}

	if withinLimit {
		// Within subscription limit → simple non-transactional create
		if err := s.repo.Create(ctx, resp); err != nil {
			return nil, err
		}
	} else {
		// Subscription limit exhausted → use credits atomically
		if s.creditSvc == nil {
			return nil, ErrInsufficientCredits
		}

		tx, err := s.repo.BeginTx(ctx)
		if err != nil {
			return nil, ErrCreditOperationFailed
		}
		defer tx.Rollback()

		// DeductTx uses FOR UPDATE row lock on users.credit_balance
		creditMeta := credit.TransactionMeta{
			RelatedEntityType: "casting",
			RelatedEntityID:   castingID,
			Description:       "apply to casting: " + cast.Title,
		}
		if err := s.creditSvc.DeductTx(ctx, tx, userID, 1, creditMeta); err != nil {
			if errors.Is(err, credit.ErrInsufficientCredits) {
				return nil, ErrInsufficientCredits
			}
			return nil, ErrCreditOperationFailed
		}

		// Create response within same transaction
		if err := s.repo.CreateTx(ctx, tx, resp); err != nil {
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, ErrCreditOperationFailed
		}
		usedCredit = true
	}

	_ = usedCredit // future: can be used for analytics or response metadata

	// Send notification to employer about new response (async with panic guard)
	if s.notifService != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Msg("[apply goroutine] recovered from panic in notification")
				}
			}()
			bgCtx := context.Background()

			model, _ := s.modelRepo.GetByID(bgCtx, prof.ID)
			modelName := "Модель"
			if model != nil && model.Name.Valid && model.Name.String != "" {
				modelName = model.Name.String
			}

			if err := s.notifService.NotifyNewResponse(
				bgCtx,
				cast.CreatorID,
				castingID,
				resp.ID,
				cast.Title,
				modelName,
			); err != nil {
				log.Error().Err(err).Str("response_id", resp.ID.String()).Msg("[apply goroutine] failed to send new response notification")
			}
		}()
	}

	return resp, nil
}

// UpdateStatus updates response status (casting owner only)
// UpdateStatus updates response status and triggers side-effects (notifications/chat).
func (s *Service) UpdateStatus(ctx context.Context, userID uuid.UUID, responseID uuid.UUID, newStatus Status) (*Response, error) {
	// Get response
	resp, err := s.repo.GetByID(ctx, responseID)
	if err != nil || resp == nil {
		return nil, ErrResponseNotFound
	}

	// Get casting to check ownership
	cast, err := s.castingRepo.GetByID(ctx, resp.CastingID)
	if err != nil || cast == nil {
		return nil, ErrCastingNotFound
	}

	// Get employer profile to check ownership
	employerProfile, err := s.employerRepo.GetByUserID(ctx, userID)
	if err != nil || employerProfile == nil {
		return nil, ErrNotCastingOwner
	}

	// Check if user owns the casting
	if cast.CreatorID != userID {
		return nil, ErrNotCastingOwner
	}

	// Check casting is still active before accepting
	if newStatus == StatusAccepted && !cast.IsActive() {
		return nil, ErrCastingNotActive
	}

	// Validate status transition (state machine)
	if !resp.CanBeUpdatedTo(newStatus) {
		return nil, ErrInvalidStatusTransition
	}

	// Get old status for refund logic
	oldStatus := resp.Status

	// Update status
	if newStatus == StatusAccepted && oldStatus != StatusAccepted {
		tx, err := s.repo.BeginTx(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()

		if err := s.repo.UpdateStatusTx(ctx, tx, responseID, newStatus); err != nil {
			return nil, err
		}

		if _, _, err := s.castingRepo.IncrementAcceptedAndMaybeCloseTx(ctx, tx, cast.ID); err != nil {
			if errors.Is(err, casting.ErrCastingFullOrClosed) {
				return nil, ErrCastingFullOrClosed
			}
			return nil, err
		}

		if err := tx.Commit(); err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.UpdateStatus(ctx, responseID, newStatus); err != nil {
			return nil, err
		}
	}

	resp.Status = newStatus
	resp.UpdatedAt = time.Now()

	// Send notification to model about status change (async with panic guard)
	if s.notifService != nil && (newStatus == StatusAccepted || newStatus == StatusRejected) {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Str("response_id", responseID.String()).Msg("[update_status goroutine] recovered from panic in notification")
				}
			}()
			bgCtx := context.Background()

			model, _ := s.modelRepo.GetByID(bgCtx, resp.ModelID)
			if model != nil {
				status := "accepted"
				if newStatus == StatusRejected {
					status = "rejected"
				}
				if err := s.notifService.NotifyResponseStatusChange(
					bgCtx,
					model.UserID,
					cast.Title,
					status,
					cast.ID,
					resp.ID,
				); err != nil {
					log.Error().Err(err).Str("response_id", responseID.String()).Msg("[update_status goroutine] failed to notify model")
				}
			}
		}()
	}

	// AUTO-CREATE CHAT ROOM ON ACCEPTANCE with context initialization
	// Immediately sends the model's offer as the first message so neither party
	// opens an empty chat — the negotiation context is visible right away.
	if s.chatSvc != nil && newStatus == StatusAccepted {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error().Interface("panic", r).Str("response_id", responseID.String()).Msg("[update_status goroutine] recovered from panic in chat creation")
				}
			}()
			bgCtx := context.Background()

			model, err := s.modelRepo.GetByID(bgCtx, resp.ModelID)
			if err != nil || model == nil {
				log.Error().Err(err).Str("model_id", resp.ModelID.String()).Msg("[update_status goroutine] failed to get model for chat creation")
				return
			}

			// Build initial context message from model's offer
			initialMsg := resp.GetMessage()
			if resp.ProposedRate.Valid {
				rateStr := fmt.Sprintf("%.0f ₸", resp.ProposedRate.Float64)
				if initialMsg != "" {
					initialMsg = initialMsg + "\n\n💰 Предложенная ставка: " + rateStr
				} else {
					initialMsg = "💰 Предложенная ставка: " + rateStr
				}
			}
			if initialMsg == "" {
				initialMsg = fmt.Sprintf("✅ Отклик на кастинг \"%s\" принят. Добро пожаловать!", cast.Title)
			}

			castingIDPtr := &cast.ID
			chatReq := &ChatRoomRequest{
				RecipientID: model.UserID,
				CastingID:   castingIDPtr,
				Message:     initialMsg,
			}

			if _, err := s.chatSvc.CreateOrGetRoom(bgCtx, userID, chatReq); err != nil {
				log.Error().Err(err).Str("response_id", responseID.String()).Msg("[update_status goroutine] failed to create/get chat room")
			}
		}()
	}

	return resp, nil
}

func isUrgentDifferentCity(cast *casting.Casting, prof *profile.ModelProfile) bool {
	if cast == nil || prof == nil || !cast.DateFrom.Valid {
		return false
	}

	if time.Until(cast.DateFrom.Time) >= 24*time.Hour {
		return false
	}

	castingCity := strings.TrimSpace(cast.City)
	modelCity := strings.TrimSpace(prof.GetCity())
	return !strings.EqualFold(castingCity, modelCity)
}

// GetByID returns response by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Response, error) {
	resp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, ErrResponseNotFound
	}
	return resp, nil
}

// ListByCasting returns responses for a casting (casting owner only)
func (s *Service) ListByCasting(ctx context.Context, userID uuid.UUID, castingID uuid.UUID, pagination *Pagination) ([]*Response, int, error) {
	// Check if user owns the casting
	cast, err := s.castingRepo.GetByID(ctx, castingID)
	if err != nil || cast == nil {
		return nil, 0, ErrCastingNotFound
	}

	// Get employer profile
	employerProfile, err := s.employerRepo.GetByUserID(ctx, userID)
	if err != nil || employerProfile == nil {
		return nil, 0, ErrNotCastingOwner
	}

	if cast.CreatorID != userID {
		return nil, 0, ErrNotCastingOwner
	}

	return s.repo.ListByCasting(ctx, castingID, pagination)
}

// ListMyApplications returns user's applications
func (s *Service) ListMyApplications(ctx context.Context, userID uuid.UUID, pagination *Pagination) ([]*Response, int, error) {
	// Get user's model profile
	prof, err := s.modelRepo.GetByUserID(ctx, userID)
	if err != nil || prof == nil {
		return nil, 0, ErrProfileRequired
	}

	return s.repo.ListByModel(ctx, prof.ID, pagination)
}

// CountMonthlyByUserID returns how many applications user made this month
func (s *Service) CountMonthlyByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.CountMonthlyByUserID(ctx, userID)
}

// CheckRequirements validates model profile against casting requirements.
// Returns a map of field→reason for each failed check. Empty map = all OK.
func CheckRequirements(cast *casting.Casting, prof *profile.ModelProfile) map[string]string {
	violations := make(map[string]string)

	// Gender
	if cast.RequiredGender.Valid && cast.RequiredGender.String != "" {
		if !prof.Gender.Valid || !strings.EqualFold(prof.Gender.String, cast.RequiredGender.String) {
			violations["gender"] = fmt.Sprintf("required %s", cast.RequiredGender.String)
		}
	}

	// Age range
	if cast.AgeMin.Valid && prof.Age.Valid && prof.Age.Int32 < cast.AgeMin.Int32 {
		violations["age_min"] = fmt.Sprintf("required min %d, got %d", cast.AgeMin.Int32, prof.Age.Int32)
	}
	if cast.AgeMax.Valid && prof.Age.Valid && prof.Age.Int32 > cast.AgeMax.Int32 {
		violations["age_max"] = fmt.Sprintf("required max %d, got %d", cast.AgeMax.Int32, prof.Age.Int32)
	}

	// Height range
	if cast.HeightMin.Valid && prof.Height.Valid && int32(prof.Height.Float64) < cast.HeightMin.Int32 {
		violations["height_min"] = fmt.Sprintf("required min %d, got %d", cast.HeightMin.Int32, int32(prof.Height.Float64))
	}
	if cast.HeightMax.Valid && prof.Height.Valid && int32(prof.Height.Float64) > cast.HeightMax.Int32 {
		violations["height_max"] = fmt.Sprintf("required max %d, got %d", cast.HeightMax.Int32, int32(prof.Height.Float64))
	}

	// Weight range
	if cast.WeightMin.Valid && prof.Weight.Valid && int32(prof.Weight.Float64) < cast.WeightMin.Int32 {
		violations["weight_min"] = fmt.Sprintf("required min %d, got %d", cast.WeightMin.Int32, int32(prof.Weight.Float64))
	}
	if cast.WeightMax.Valid && prof.Weight.Valid && int32(prof.Weight.Float64) > cast.WeightMax.Int32 {
		violations["weight_max"] = fmt.Sprintf("required max %d, got %d", cast.WeightMax.Int32, int32(prof.Weight.Float64))
	}

	// Hair color (model's hair color must be in the casting's required list)
	if len(cast.RequiredHairColors) > 0 && prof.HairColor.Valid {
		if !containsIgnoreCase([]string(cast.RequiredHairColors), prof.HairColor.String) {
			violations["hair_color"] = fmt.Sprintf("required one of %v, got %s", []string(cast.RequiredHairColors), prof.HairColor.String)
		}
	}

	// Eye color
	if len(cast.RequiredEyeColors) > 0 && prof.EyeColor.Valid {
		if !containsIgnoreCase([]string(cast.RequiredEyeColors), prof.EyeColor.String) {
			violations["eye_color"] = fmt.Sprintf("required one of %v, got %s", []string(cast.RequiredEyeColors), prof.EyeColor.String)
		}
	}

	// Clothing size
	if len(cast.ClothingSizes) > 0 && prof.ClothingSize.Valid {
		if !containsIgnoreCase([]string(cast.ClothingSizes), prof.ClothingSize.String) {
			violations["clothing_size"] = fmt.Sprintf("required one of %v, got %s", []string(cast.ClothingSizes), prof.ClothingSize.String)
		}
	}

	// Shoe size
	if len(cast.ShoeSizes) > 0 && prof.ShoeSize.Valid {
		if !containsIgnoreCase([]string(cast.ShoeSizes), prof.ShoeSize.String) {
			violations["shoe_size"] = fmt.Sprintf("required one of %v, got %s", []string(cast.ShoeSizes), prof.ShoeSize.String)
		}
	}

	return violations
}

// containsIgnoreCase checks if a string is in a slice (case-insensitive).
func containsIgnoreCase(list []string, value string) bool {
	for _, item := range list {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}
