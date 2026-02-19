package casting

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

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
	if req.AgeMin != nil && req.AgeMax != nil && *req.AgeMin > *req.AgeMax {
		errs["age_min"] = "age_min must be <= age_max"
	}
	if req.HeightMin != nil && req.HeightMax != nil && *req.HeightMin > *req.HeightMax {
		errs["height_min"] = "height_min must be <= height_max"
	}
	if req.WeightMin != nil && req.WeightMax != nil && *req.WeightMin > *req.WeightMax {
		errs["weight_min"] = "weight_min must be <= weight_max"
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

// applyRequirementsToCreate maps flat request fields to the Casting entity
func applyRequirementsToCreate(casting *Casting, req *CreateCastingRequest) {
	if req.RequiredGender != "" {
		casting.RequiredGender = sql.NullString{String: req.RequiredGender, Valid: true}
	}
	if req.AgeMin != nil {
		casting.AgeMin = sql.NullInt32{Int32: int32(*req.AgeMin), Valid: true}
	}
	if req.AgeMax != nil {
		casting.AgeMax = sql.NullInt32{Int32: int32(*req.AgeMax), Valid: true}
	}
	if req.HeightMin != nil {
		casting.HeightMin = sql.NullInt32{Int32: int32(*req.HeightMin), Valid: true}
	}
	if req.HeightMax != nil {
		casting.HeightMax = sql.NullInt32{Int32: int32(*req.HeightMax), Valid: true}
	}
	if req.WeightMin != nil {
		casting.WeightMin = sql.NullInt32{Int32: int32(*req.WeightMin), Valid: true}
	}
	if req.WeightMax != nil {
		casting.WeightMax = sql.NullInt32{Int32: int32(*req.WeightMax), Valid: true}
	}
	if req.RequiredExperience != "" {
		casting.RequiredExperience = sql.NullString{String: req.RequiredExperience, Valid: true}
	}
	if len(req.RequiredLanguages) > 0 {
		casting.RequiredLanguages = pq.StringArray(req.RequiredLanguages)
	}
	if len(req.ClothingSizes) > 0 {
		casting.ClothingSizes = pq.StringArray(req.ClothingSizes)
	}
	if len(req.ShoeSizes) > 0 {
		casting.ShoeSizes = pq.StringArray(req.ShoeSizes)
	}
	if len(req.Tags) > 0 {
		casting.Tags = pq.StringArray(req.Tags)
	}
	if req.WorkType != "" {
		casting.WorkType = sql.NullString{String: req.WorkType, Valid: true}
	}
	if req.EventLocation != "" {
		casting.EventLocation = sql.NullString{String: req.EventLocation, Valid: true}
	}
	casting.IsUrgent = req.IsUrgent
}

// applyRequirementsToUpdate maps flat update request fields to the Casting entity
func applyRequirementsToUpdate(casting *Casting, req *UpdateCastingRequest) {
	if req.RequiredGender != "" {
		casting.RequiredGender = sql.NullString{String: req.RequiredGender, Valid: true}
	}
	if req.AgeMin != nil {
		casting.AgeMin = sql.NullInt32{Int32: int32(*req.AgeMin), Valid: true}
	}
	if req.AgeMax != nil {
		casting.AgeMax = sql.NullInt32{Int32: int32(*req.AgeMax), Valid: true}
	}
	if req.HeightMin != nil {
		casting.HeightMin = sql.NullInt32{Int32: int32(*req.HeightMin), Valid: true}
	}
	if req.HeightMax != nil {
		casting.HeightMax = sql.NullInt32{Int32: int32(*req.HeightMax), Valid: true}
	}
	if req.WeightMin != nil {
		casting.WeightMin = sql.NullInt32{Int32: int32(*req.WeightMin), Valid: true}
	}
	if req.WeightMax != nil {
		casting.WeightMax = sql.NullInt32{Int32: int32(*req.WeightMax), Valid: true}
	}
	if req.RequiredExperience != "" {
		casting.RequiredExperience = sql.NullString{String: req.RequiredExperience, Valid: true}
	}
	if req.RequiredLanguages != nil {
		casting.RequiredLanguages = pq.StringArray(req.RequiredLanguages)
	}
	if req.ClothingSizes != nil {
		casting.ClothingSizes = pq.StringArray(req.ClothingSizes)
	}
	if req.ShoeSizes != nil {
		casting.ShoeSizes = pq.StringArray(req.ShoeSizes)
	}
	if req.Tags != nil {
		casting.Tags = pq.StringArray(req.Tags)
	}
	if req.WorkType != "" {
		casting.WorkType = sql.NullString{String: req.WorkType, Valid: true}
	}
	if req.EventLocation != "" {
		casting.EventLocation = sql.NullString{String: req.EventLocation, Valid: true}
	}
	if req.IsUrgent != nil {
		casting.IsUrgent = *req.IsUrgent
	}
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

	eventDatetime, err := parseRFC3339Field(req.EventDatetime)
	if err != nil {
		return nil, ErrInvalidDateFromFormat
	}
	deadlineAt, err := parseRFC3339Field(req.DeadlineAt)
	if err != nil {
		return nil, ErrInvalidDateToFormat
	}

	// Check if user is employer
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrCastingNotFound
	}
	if !u.CanCreateCasting() {
		return nil, ErrOnlyEmployersCanCreate
	}
	requestedStatus := StatusActive
	if req.Status != "" {
		requestedStatus = Status(req.Status)
	}

	if (u.Role == user.RoleEmployer || u.Role == user.RoleAgency) && !u.IsVerificationApproved() && requestedStatus != StatusDraft {
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
		Status:        requestedStatus,
		IsPromoted:    false,
		ViewCount:     0,
		ResponseCount: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
		DateFrom:      dateFrom,
		DateTo:        dateTo,
		EventDatetime: eventDatetime,
		DeadlineAt:    deadlineAt,
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
	if req.PayMin != nil {
		casting.PayMin = sql.NullFloat64{Float64: *req.PayMin, Valid: true}
	}
	if req.PayMax != nil {
		casting.PayMax = sql.NullFloat64{Float64: *req.PayMax, Valid: true}
	}
	if req.CoverImageURL != "" {
		casting.CoverImageURL = sql.NullString{String: req.CoverImageURL, Valid: true}
	}

	// Apply requirements from flat fields
	applyRequirementsToCreate(casting, req)

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

	// Validate pay range
	newPayMin := casting.PayMin
	newPayMax := casting.PayMax
	if req.PayMin != nil {
		newPayMin = sql.NullFloat64{Float64: *req.PayMin, Valid: true}
	}
	if req.PayMax != nil {
		newPayMax = sql.NullFloat64{Float64: *req.PayMax, Valid: true}
	}
	if newPayMin.Valid && newPayMax.Valid && newPayMin.Float64 > newPayMax.Float64 {
		return nil, ValidationErrors{"pay_min": "pay_min must be <= pay_max"}
	}

	// Validate requirements ranges
	if req.AgeMin != nil && req.AgeMax != nil && *req.AgeMin > *req.AgeMax {
		return nil, ValidationErrors{"age_min": "age_min must be <= age_max"}
	}
	if req.HeightMin != nil && req.HeightMax != nil && *req.HeightMin > *req.HeightMax {
		return nil, ValidationErrors{"height_min": "height_min must be <= height_max"}
	}
	if req.WeightMin != nil && req.WeightMax != nil && *req.WeightMin > *req.WeightMax {
		return nil, ValidationErrors{"weight_min": "weight_min must be <= weight_max"}
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

	if req.EventDatetime != nil {
		t, err := parseRFC3339Field(req.EventDatetime)
		if err != nil {
			return nil, ErrInvalidDateFromFormat
		}
		casting.EventDatetime = t
	}
	if req.DeadlineAt != nil {
		t, err := parseRFC3339Field(req.DeadlineAt)
		if err != nil {
			return nil, ErrInvalidDateToFormat
		}
		casting.DeadlineAt = t
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
	casting.PayMin = newPayMin
	casting.PayMax = newPayMax
	if req.PayType != "" {
		casting.PayType = req.PayType
	}
	if req.CoverImageURL != "" {
		casting.CoverImageURL = sql.NullString{String: req.CoverImageURL, Valid: true}
	}

	// Apply requirements from flat fields
	applyRequirementsToUpdate(casting, req)

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
