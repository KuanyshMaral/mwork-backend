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

func validateCreateCastingRequest(req *CreateCastingRequest) ValidationErrors {
	errs := ValidationErrors{}

	if req.PayMin != nil && req.PayMax != nil && *req.PayMin > *req.PayMax {
		errs["pay_min"] = "pay_min must be <= pay_max"
	}

	if reqErrs := validateRequirementsRange(req.Requirements); reqErrs != nil {
		for k, v := range reqErrs {
			errs[k] = v
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

func validateRequirementsRange(req *RequirementsRequest) ValidationErrors {
	if req == nil {
		return nil
	}

	errs := ValidationErrors{}
	if req.AgeMin != nil && req.AgeMax != nil && *req.AgeMin > *req.AgeMax {
		errs["requirements.age_min"] = "requirements.age_min must be <= requirements.age_max"
	}
	if req.HeightMin != nil && req.HeightMax != nil && *req.HeightMin > *req.HeightMax {
		errs["requirements.height_min"] = "requirements.height_min must be <= requirements.height_max"
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func validateStatusTransition(current Status, next Status) error {
	if current == next {
		return nil
	}

	switch current {
	case StatusDraft:
		if next == StatusActive || next == StatusClosed {
			return nil
		}
	case StatusActive:
		if next == StatusClosed {
			return nil
		}
	case StatusClosed:
		// Closed is terminal and cannot be reopened.
	}

	return ErrInvalidStatusTransition
}

func parseRFC3339Field(value *string) (sql.NullTime, error) {
	if value == nil {
		return sql.NullTime{}, nil
	}

	t, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return sql.NullTime{}, err
	}

	return sql.NullTime{Time: t, Valid: true}, nil
}

// Create creates a new casting
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req *CreateCastingRequest) (*Casting, error) {
	if verr := validateCreateCastingRequest(req); verr != nil {
		return nil, verr
	}

	dateFrom, err := parseRFC3339Field(req.DateFrom)
	if err != nil {
		return nil, ErrInvalidDateFromFormat
	}
	dateTo, err := parseRFC3339Field(req.DateTo)
	if err != nil {
		return nil, ErrInvalidDateToFormat
	}
	if dateFrom.Valid && dateTo.Valid && dateFrom.Time.After(dateTo.Time) {
		return nil, ErrInvalidDateRange
	}

	// Check if user is employer
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrCastingNotFound
	}
	if !u.CanCreateCasting() {
		return nil, ErrOnlyEmployersCanCreate
	}
	if (u.Role == user.RoleEmployer || u.Role == user.RoleAgency) && !u.IsVerificationApproved() {
		return nil, ErrEmployerNotVerified
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
		DateFrom:      dateFrom,
		DateTo:        dateTo,
	}

	if u.IsCompanyVerified() {
		casting.ModerationStatus = ModerationApproved
	} else {
		casting.ModerationStatus = ModerationPending
	}

	if req.Address != "" {
		casting.Address = sql.NullString{String: req.Address, Valid: true}
	}

	if req.PayType != "" {
		casting.PayType = req.PayType
	}

	if req.CoverImageURL != "" {
		casting.CoverImageURL = sql.NullString{String: req.CoverImageURL, Valid: true}
	}

	if req.Requirements != nil {
		reqData := Requirements{
			Gender:             req.Requirements.Gender,
			ExperienceRequired: req.Requirements.ExperienceRequired,
			Languages:          req.Requirements.Languages,
		}
		if req.Requirements.AgeMin != nil {
			reqData.AgeMin = *req.Requirements.AgeMin
		}
		if req.Requirements.AgeMax != nil {
			reqData.AgeMax = *req.Requirements.AgeMax
		}
		if req.Requirements.HeightMin != nil {
			reqData.HeightMin = *req.Requirements.HeightMin
		}
		if req.Requirements.HeightMax != nil {
			reqData.HeightMax = *req.Requirements.HeightMax
		}
		casting.Requirements, _ = json.Marshal(reqData)
	} else {
		casting.Requirements, _ = json.Marshal(Requirements{})
	}

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

	if req.PayMin != nil {
		casting.PayMin = sql.NullFloat64{Float64: *req.PayMin, Valid: true}
	}
	if req.PayMax != nil {
		casting.PayMax = sql.NullFloat64{Float64: *req.PayMax, Valid: true}
	}
	if casting.PayMin.Valid && casting.PayMax.Valid && casting.PayMin.Float64 > casting.PayMax.Float64 {
		return nil, ValidationErrors{"pay_min": "pay_min must be <= pay_max"}
	}

	if reqErrs := validateRequirementsRange(req.Requirements); reqErrs != nil {
		return nil, reqErrs
	}

	if req.DateFrom != nil {
		t, err := parseRFC3339Field(req.DateFrom)
		if err != nil {
			return nil, ErrInvalidDateFromFormat
		}
		casting.DateFrom = t
	}
	if req.DateTo != nil {
		t, err := parseRFC3339Field(req.DateTo)
		if err != nil {
			return nil, ErrInvalidDateToFormat
		}
		casting.DateTo = t
	}
	if casting.DateFrom.Valid && casting.DateTo.Valid && casting.DateFrom.Time.After(casting.DateTo.Time) {
		return nil, ErrInvalidDateRange
	}

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
	if req.CoverImageURL != "" {
		casting.CoverImageURL = sql.NullString{String: req.CoverImageURL, Valid: true}
	}

	if req.Requirements != nil {
		reqData := Requirements{
			Gender:             req.Requirements.Gender,
			ExperienceRequired: req.Requirements.ExperienceRequired,
			Languages:          req.Requirements.Languages,
		}
		if req.Requirements.AgeMin != nil {
			reqData.AgeMin = *req.Requirements.AgeMin
		}
		if req.Requirements.AgeMax != nil {
			reqData.AgeMax = *req.Requirements.AgeMax
		}
		if req.Requirements.HeightMin != nil {
			reqData.HeightMin = *req.Requirements.HeightMin
		}
		if req.Requirements.HeightMax != nil {
			reqData.HeightMax = *req.Requirements.HeightMax
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

	if err := validateStatusTransition(casting.Status, status); err != nil {
		return nil, err
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

// ListByCreator returns castings by creator
func (s *Service) ListByCreator(ctx context.Context, creatorID uuid.UUID, pagination *Pagination) ([]*Casting, int, error) {
	return s.repo.ListByCreator(ctx, creatorID, pagination)
}
