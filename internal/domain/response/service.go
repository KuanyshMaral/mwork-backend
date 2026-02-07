package response

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/casting"
	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/domain/profile"
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
	repo         Repository
	castingRepo  casting.Repository
	modelRepo    profile.ModelRepository
	employerRepo profile.EmployerRepository
	notifService NotificationService
	creditSvc    credit.Service       // ✅ FIXED: Using standard credit.Service interface
	chatSvc      ChatServiceInterface // Chat service for auto-creating rooms on acceptance
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
func (s *Service) SetCreditService(creditSvc credit.Service) { // ✅ FIXED: Using credit.Service
	s.creditSvc = creditSvc
}

// SetChatService sets the chat service (optional, to avoid circular dependency)
func (s *Service) SetChatService(chatSvc ChatServiceInterface) {
	s.chatSvc = chatSvc
}

// Apply applies to a casting
// B1: Credit deduction before creating response
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

	// B1: DEDUCT CREDIT BEFORE CREATING RESPONSE
	// This ensures no response is created without payment
	if s.creditSvc != nil {
		creditMeta := credit.TransactionMeta{ // ✅ FIXED: Using credit.TransactionMeta
			RelatedEntityType: "casting",
			RelatedEntityID:   castingID,
			Description:       fmt.Sprintf("Applied to casting %s", cast.Title),
		}

		err := s.creditSvc.Deduct(ctx, userID, 1, creditMeta)
		if err != nil {
			// Check if it's insufficient credits error
			if errors.Is(err, credit.ErrInsufficientCredits) { // ✅ FIXED: Using credit.ErrInsufficientCredits
				return nil, ErrInsufficientCredits
			}
			return nil, err
		}
	}

	now := time.Now()
	response := &Response{
		ID:        uuid.New(),
		CastingID: castingID,
		ModelID:   prof.ID,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if req.Message != "" {
		response.Message = sql.NullString{String: req.Message, Valid: true}
	}
	if req.ProposedRate != nil {
		response.ProposedRate = sql.NullFloat64{Float64: *req.ProposedRate, Valid: true}
	}

	// Create response
	err = s.repo.Create(ctx, response)
	if err != nil {
		// B1: AUTOMATIC ROLLBACK - Refund credit if response creation fails
		if s.creditSvc != nil {
			refundMeta := credit.TransactionMeta{ // ✅ FIXED: Using credit.TransactionMeta
				RelatedEntityType: "response",
				RelatedEntityID:   response.ID,
				Description:       fmt.Sprintf("Automatic rollback refund for response %s", response.ID.String()),
			}
			// Use background context to ensure refund completes even if request is cancelled
			bgCtx := context.Background()
			_ = s.creditSvc.Add(bgCtx, userID, 1, credit.TransactionTypeRefund, refundMeta) // ✅ FIXED: Using credit.TransactionTypeRefund
		}
		return nil, err
	}

	// Send notification to employer about new response
	if s.notifService != nil {
		go func() {
			// Use background context to avoid cancellation
			bgCtx := context.Background()

			// Get model details for notification
			model, _ := s.modelRepo.GetByID(bgCtx, prof.ID)
			modelName := "Модель"
			if model != nil && model.Name.Valid && model.Name.String != "" {
				modelName = model.Name.String
			}

			// Send notification
			_ = s.notifService.NotifyNewResponse(
				bgCtx,
				cast.CreatorID,
				castingID,
				response.ID,
				cast.Title,
				modelName,
			)
		}()
	}

	return response, nil
}

// UpdateStatus updates response status (casting owner only)
// B2: Refund on rejected status with idempotency
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

	// Validate status transition
	if !resp.CanBeUpdatedTo(newStatus) {
		return nil, ErrInvalidStatusTransition
	}

	// Get old status for refund logic
	oldStatus := resp.Status

	// Update status
	if err := s.repo.UpdateStatus(ctx, responseID, newStatus); err != nil {
		return nil, err
	}

	resp.Status = newStatus
	resp.UpdatedAt = time.Now()

	// B2: REFUND ON REJECTION WITH IDEMPOTENCY
	// Only refund when transitioning to rejected status
	if s.creditSvc != nil && newStatus == StatusRejected && oldStatus == StatusPending {
		// Check idempotency - ensure we don't refund twice
		hasRefund, err := s.creditSvc.HasRefund(ctx, responseID) // ✅ FIXED: Using standard HasRefund method
		if err == nil && !hasRefund {
			// Get model's user ID for refund
			model, err := s.modelRepo.GetByID(ctx, resp.ModelID)
			if err == nil && model != nil {
				refundMeta := credit.TransactionMeta{ // ✅ FIXED: Using credit.TransactionMeta
					RelatedEntityType: "response",
					RelatedEntityID:   responseID,
					Description:       fmt.Sprintf("Refund due to rejection for response %s", responseID.String()),
				}

				// Refund 1 credit
				_ = s.creditSvc.Add(ctx, model.UserID, 1, credit.TransactionTypeRefund, refundMeta) // ✅ FIXED: Using credit.TransactionTypeRefund
			}
		}
	}

	// Send notification to model about status change
	if s.notifService != nil && (newStatus == StatusAccepted || newStatus == StatusRejected) {
		go func() {
			// Use background context to avoid cancellation
			bgCtx := context.Background()

			// Get model's user ID
			model, _ := s.modelRepo.GetByID(bgCtx, resp.ModelID)
			if model != nil {
				status := "accepted"
				if newStatus == StatusRejected {
					status = "rejected"
				}

				_ = s.notifService.NotifyResponseStatusChange(
					bgCtx,
					model.UserID,
					cast.Title,
					status,
					cast.ID,
					resp.ID,
				)
			}
		}()
	}

	// Task 1: AUTO-CREATE CHAT ROOM ON ACCEPTANCE
	// When response is accepted, automatically create a chat room between employer and model
	if s.chatSvc != nil && newStatus == StatusAccepted {
		go func() {
			// Use background context to avoid cancellation
			bgCtx := context.Background()

			// Get model's user ID
			model, err := s.modelRepo.GetByID(bgCtx, resp.ModelID)
			if err != nil || model == nil {
				return
			}

			// Create chat room request using proper DTO
			castingIDPtr := &cast.ID
			chatReq := &ChatRoomRequest{
				RecipientID: model.UserID,
				CastingID:   castingIDPtr,
			}

			// Create or get existing room between employer and model
			_, _ = s.chatSvc.CreateOrGetRoom(bgCtx, userID, chatReq)
		}()
	}

	return resp, nil
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
