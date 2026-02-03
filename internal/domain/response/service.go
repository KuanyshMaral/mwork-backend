package response

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/casting"
	"github.com/mwork/mwork-api/internal/domain/profile"
)

// NotificationService interface for notification operations
type NotificationService interface {
	NotifyNewResponse(ctx context.Context, employerUserID uuid.UUID, castingID uuid.UUID, responseID uuid.UUID, castingTitle string, modelName string) error
	NotifyResponseStatusChange(ctx context.Context, modelUserID uuid.UUID, castingTitle string, status string, castingID uuid.UUID, responseID uuid.UUID) error
}

// Service handles response business logic
type Service struct {
	repo         Repository
	castingRepo  casting.Repository
	modelRepo    profile.ModelRepository
	employerRepo profile.EmployerRepository
	notifService NotificationService
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

// Apply applies to a casting
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

	if err := s.repo.Create(ctx, response); err != nil {
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

	// Update status
	if err := s.repo.UpdateStatus(ctx, responseID, newStatus); err != nil {
		return nil, err
	}

	resp.Status = newStatus
	resp.UpdatedAt = time.Now()

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
