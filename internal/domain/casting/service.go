package casting

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

// NotificationService interface for notification operations
type NotificationService interface {
	NotifyAgencyFollowersNewCasting(ctx context.Context, organizationID uuid.UUID, castingID uuid.UUID, castingTitle string) error
}

// Service handles casting business logic
type Service struct {
	repo         Repository
	userRepo     user.Repository
	notifService NotificationService
}

// NewService creates casting service
func NewService(repo Repository, userRepo user.Repository) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
	}
}

// SetNotificationService sets the notification service (optional)
func (s *Service) SetNotificationService(notifService NotificationService) {
	s.notifService = notifService
}

// Create creates a new casting
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req *CreateCastingRequest) (*Casting, error) {
	// Check if user is employer
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrCastingNotFound
	}
	if !u.CanCreateCasting() {
		return nil, ErrOnlyEmployersCanCreate
	}

	now := time.Now()
	casting := &Casting{
		ID:            uuid.New(),
		CreatorID:     userID,
		Title:         req.Title,
		Description:   req.Description,
		City:          req.City,
		PayType:       "negotiable",
		Status:        StatusActive,
		IsPromoted:    false,
		ViewCount:     0,
		ResponseCount: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Address
	if req.Address != "" {
		casting.Address = sql.NullString{String: req.Address, Valid: true}
	}

	// Payment
	if req.PayMin != nil {
		casting.PayMin = sql.NullFloat64{Float64: *req.PayMin, Valid: true}
	}
	if req.PayMax != nil {
		casting.PayMax = sql.NullFloat64{Float64: *req.PayMax, Valid: true}
	}
	if req.PayType != "" {
		casting.PayType = req.PayType
	}

	// Dates
	if req.DateFrom != nil {
		if t, err := time.Parse(time.RFC3339, *req.DateFrom); err == nil {
			casting.DateFrom = sql.NullTime{Time: t, Valid: true}
		}
	}
	if req.DateTo != nil {
		if t, err := time.Parse(time.RFC3339, *req.DateTo); err == nil {
			casting.DateTo = sql.NullTime{Time: t, Valid: true}
		}
	}

	// Cover image URL
	if req.CoverImageURL != "" {
		casting.CoverImageURL = sql.NullString{String: req.CoverImageURL, Valid: true}
	}

	// Requirements (JSONB)
	if req.Requirements != nil {
		reqData := Requirements{
			Gender:             req.Requirements.Gender,
			AgeMin:             req.Requirements.AgeMin,
			AgeMax:             req.Requirements.AgeMax,
			HeightMin:          req.Requirements.HeightMin,
			HeightMax:          req.Requirements.HeightMax,
			ExperienceRequired: req.Requirements.ExperienceRequired,
			Languages:          req.Requirements.Languages,
		}
		casting.Requirements, _ = json.Marshal(reqData)
	} else {
		casting.Requirements, _ = json.Marshal(Requirements{})
	}

	// Status
	if req.Status != "" {
		casting.Status = Status(req.Status)
	}

	if err := s.repo.Create(ctx, casting); err != nil {
		return nil, err
	}

	return casting, nil
}

// GetByID returns casting by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Casting, error) {
	casting, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if casting == nil {
		return nil, ErrCastingNotFound
	}
	return casting, nil
}

// Update updates casting
func (s *Service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateCastingRequest) (*Casting, error) {
	casting, err := s.repo.GetByID(ctx, id)
	if err != nil || casting == nil {
		return nil, ErrCastingNotFound
	}

	if !casting.CanBeEditedBy(userID) {
		return nil, ErrNotCastingOwner
	}

	// Update fields
	if req.Title != "" {
		casting.Title = req.Title
	}
	if req.Description != "" {
		casting.Description = req.Description
	}
	if req.City != "" {
		casting.City = req.City
	}
	if req.Address != "" {
		casting.Address = sql.NullString{String: req.Address, Valid: true}
	}
	if req.PayMin != nil {
		casting.PayMin = sql.NullFloat64{Float64: *req.PayMin, Valid: true}
	}
	if req.PayMax != nil {
		casting.PayMax = sql.NullFloat64{Float64: *req.PayMax, Valid: true}
	}
	if req.PayType != "" {
		casting.PayType = req.PayType
	}
	if req.DateFrom != nil {
		if t, err := time.Parse(time.RFC3339, *req.DateFrom); err == nil {
			casting.DateFrom = sql.NullTime{Time: t, Valid: true}
		}
	}
	if req.DateTo != nil {
		if t, err := time.Parse(time.RFC3339, *req.DateTo); err == nil {
			casting.DateTo = sql.NullTime{Time: t, Valid: true}
		}
	}
	if req.CoverImageURL != "" {
		casting.CoverImageURL = sql.NullString{String: req.CoverImageURL, Valid: true}
	}

	// Update requirements
	if req.Requirements != nil {
		reqData := Requirements{
			Gender:             req.Requirements.Gender,
			AgeMin:             req.Requirements.AgeMin,
			AgeMax:             req.Requirements.AgeMax,
			HeightMin:          req.Requirements.HeightMin,
			HeightMax:          req.Requirements.HeightMax,
			ExperienceRequired: req.Requirements.ExperienceRequired,
			Languages:          req.Requirements.Languages,
		}
		casting.Requirements, _ = json.Marshal(reqData)
	}

	casting.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, casting); err != nil {
		return nil, err
	}

	return casting, nil
}

// UpdateStatus updates casting status
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, userID uuid.UUID, status Status) (*Casting, error) {
	casting, err := s.repo.GetByID(ctx, id)
	if err != nil || casting == nil {
		return nil, ErrCastingNotFound
	}

	if !casting.CanBeEditedBy(userID) {
		return nil, ErrNotCastingOwner
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return nil, err
	}

	casting.Status = status
	return casting, nil
}

// Delete soft-deletes casting
func (s *Service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	casting, err := s.repo.GetByID(ctx, id)
	if err != nil || casting == nil {
		return ErrCastingNotFound
	}

	if !casting.CanBeEditedBy(userID) {
		return ErrNotCastingOwner
	}

	return s.repo.Delete(ctx, id)
}

// List returns castings with filters
func (s *Service) List(ctx context.Context, filter *Filter, sortBy SortBy, pagination *Pagination) ([]*Casting, int, error) {
	return s.repo.List(ctx, filter, sortBy, pagination)
}

// IncrementViewCount increments view count
func (s *Service) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	return s.repo.IncrementViewCount(ctx, id)
}
