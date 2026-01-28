package profile

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

// Service handles profile business logic
type Service struct {
	modelRepo    ModelRepository
	employerRepo EmployerRepository
	userRepo     user.Repository
}

// NewService creates profile service
func NewService(modelRepo ModelRepository, employerRepo EmployerRepository, userRepo user.Repository) *Service {
	return &Service{
		modelRepo:    modelRepo,
		employerRepo: employerRepo,
		userRepo:     userRepo,
	}
}

// CreateModelProfile creates a new model profile
func (s *Service) CreateModelProfile(ctx context.Context, userID uuid.UUID, req *CreateModelProfileRequest) (*ModelProfile, error) {
	// Check if user exists and is a model
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrProfileNotFound
	}
	if !u.IsModel() {
		return nil, ErrInvalidProfileType
	}

	// Check if profile already exists
	existing, _ := s.modelRepo.GetByUserID(ctx, userID)
	if existing != nil {
		return nil, ErrProfileAlreadyExists
	}

	now := time.Now()
	profile := &ModelProfile{
		ID:           uuid.New(),
		UserID:       userID,
		IsPublic:     true,
		ProfileViews: 0,
		Rating:       0,
		TotalReviews: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Set fields from request
	if req.Name != "" {
		profile.Name = sql.NullString{String: req.Name, Valid: true}
	}
	if req.Bio != "" {
		profile.Bio = sql.NullString{String: req.Bio, Valid: true}
	}
	if req.City != "" {
		profile.City = sql.NullString{String: req.City, Valid: true}
	}
	if req.Age != nil {
		profile.Age = sql.NullInt32{Int32: int32(*req.Age), Valid: true}
	}
	if req.Height != nil {
		profile.Height = sql.NullFloat64{Float64: *req.Height, Valid: true}
	}
	if req.Weight != nil {
		profile.Weight = sql.NullFloat64{Float64: *req.Weight, Valid: true}
	}
	if req.Gender != "" {
		profile.Gender = sql.NullString{String: req.Gender, Valid: true}
	}
	if req.HourlyRate != nil {
		profile.HourlyRate = sql.NullFloat64{Float64: *req.HourlyRate, Valid: true}
	}
	if req.Experience != nil {
		profile.Experience = sql.NullInt32{Int32: int32(*req.Experience), Valid: true}
	}

	profile.SetLanguages(req.Languages)
	profile.SetCategories(req.Categories)

	if err := s.modelRepo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// CreateEmployerProfile creates a new employer profile
func (s *Service) CreateEmployerProfile(ctx context.Context, userID uuid.UUID, req *CreateEmployerProfileRequest) (*EmployerProfile, error) {
	// Check if user exists and is employer
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrProfileNotFound
	}
	if !u.IsEmployer() {
		return nil, ErrInvalidProfileType
	}

	// Check if profile already exists
	existing, _ := s.employerRepo.GetByUserID(ctx, userID)
	if existing != nil {
		return nil, ErrProfileAlreadyExists
	}

	now := time.Now()
	profile := &EmployerProfile{
		ID:             uuid.New(),
		UserID:         userID,
		CompanyName:    req.CompanyName,
		Rating:         0,
		TotalReviews:   0,
		CastingsPosted: 0,
		IsVerified:     false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if req.CompanyType != "" {
		profile.CompanyType = sql.NullString{String: req.CompanyType, Valid: true}
	}
	if req.Description != "" {
		profile.Description = sql.NullString{String: req.Description, Valid: true}
	}
	if req.City != "" {
		profile.City = sql.NullString{String: req.City, Valid: true}
	}
	if req.ContactPerson != "" {
		profile.ContactPerson = sql.NullString{String: req.ContactPerson, Valid: true}
	}
	if req.ContactPhone != "" {
		profile.ContactPhone = sql.NullString{String: req.ContactPhone, Valid: true}
	}

	if err := s.employerRepo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// GetModelProfileByID returns model profile by ID
func (s *Service) GetModelProfileByID(ctx context.Context, id uuid.UUID) (*ModelProfile, error) {
	profile, err := s.modelRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

// GetModelProfileByUserID returns model profile by user ID
func (s *Service) GetModelProfileByUserID(ctx context.Context, userID uuid.UUID) (*ModelProfile, error) {
	profile, err := s.modelRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

// GetEmployerProfileByID returns employer profile by ID
func (s *Service) GetEmployerProfileByID(ctx context.Context, id uuid.UUID) (*EmployerProfile, error) {
	profile, err := s.employerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

// GetEmployerProfileByUserID returns employer profile by user ID
func (s *Service) GetEmployerProfileByUserID(ctx context.Context, userID uuid.UUID) (*EmployerProfile, error) {
	profile, err := s.employerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

// UpdateModelProfile updates model profile
func (s *Service) UpdateModelProfile(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateModelProfileRequest) (*ModelProfile, error) {
	profile, err := s.modelRepo.GetByID(ctx, id)
	if err != nil || profile == nil {
		return nil, ErrProfileNotFound
	}

	if profile.UserID != userID {
		return nil, ErrNotProfileOwner
	}

	// Update fields
	if req.Name != "" {
		profile.Name = sql.NullString{String: req.Name, Valid: true}
	}
	if req.Bio != "" {
		profile.Bio = sql.NullString{String: req.Bio, Valid: true}
	}
	if req.City != "" {
		profile.City = sql.NullString{String: req.City, Valid: true}
	}
	if req.Age != nil {
		profile.Age = sql.NullInt32{Int32: int32(*req.Age), Valid: true}
	}
	if req.Height != nil {
		profile.Height = sql.NullFloat64{Float64: *req.Height, Valid: true}
	}
	if req.Weight != nil {
		profile.Weight = sql.NullFloat64{Float64: *req.Weight, Valid: true}
	}
	if req.Gender != "" {
		profile.Gender = sql.NullString{String: req.Gender, Valid: true}
	}
	if req.IsPublic != nil {
		profile.IsPublic = *req.IsPublic
	}
	if req.Visibility != "" {
		profile.Visibility = sql.NullString{String: req.Visibility, Valid: true}
	}
	if req.Languages != nil {
		profile.SetLanguages(req.Languages)
	}
	if req.Categories != nil {
		profile.SetCategories(req.Categories)
	}

	profile.UpdatedAt = time.Now()

	if err := s.modelRepo.Update(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// ListModels returns model profiles with filters
func (s *Service) ListModels(ctx context.Context, filter *Filter, pagination *Pagination) ([]*ModelProfile, int, error) {
	return s.modelRepo.List(ctx, filter, pagination)
}

// IncrementModelViewCount increments mo view count
func (s *Service) IncrementModelViewCount(ctx context.Context, id uuid.UUID) error {
	return s.modelRepo.IncrementViewCount(ctx, id)
}
