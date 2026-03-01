package profile

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

// Service handles profile business logic
type Service struct {
	modelRepo     ModelRepository
	employerRepo  EmployerRepository
	adminRepo     AdminRepository
	userRepo      user.Repository
	uploadBaseURL string
}

// NewService creates profile service
func NewService(modelRepo ModelRepository, employerRepo EmployerRepository, adminRepo AdminRepository, userRepo user.Repository, uploadBaseURL string) *Service {
	return &Service{
		modelRepo:     modelRepo,
		employerRepo:  employerRepo,
		adminRepo:     adminRepo,
		userRepo:      userRepo,
		uploadBaseURL: uploadBaseURL,
	}
}

func (s *Service) buildAvatarURL(filePath sql.NullString) string {
	if !filePath.Valid || filePath.String == "" {
		return ""
	}
	// Assuming storage.GetURL logic is basically concatenating base + path,
	// but since we only have baseURL here, we construct it. This aligns with standard upload usage.
	// If the file path already starts with http, return it directly.
	if len(filePath.String) >= 4 && filePath.String[:4] == "http" {
		return filePath.String
	}
	return s.uploadBaseURL + "/" + filePath.String
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
	profile.SetSkills(req.Skills)
	profile.SetTravelCities(req.TravelCities)

	if req.BarterAccepted != nil {
		profile.BarterAccepted = *req.BarterAccepted
	}
	if req.AcceptRemoteWork != nil {
		profile.AcceptRemoteWork = *req.AcceptRemoteWork
	}
	if req.Visibility != "" {
		profile.Visibility = sql.NullString{String: req.Visibility, Valid: true}
	}

	if err := s.modelRepo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// EnsureModelProfile creates an empty model profile if it doesn't exist.
func (s *Service) EnsureModelProfile(ctx context.Context, userID uuid.UUID) (*ModelProfile, error) {
	existing, err := s.modelRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
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
	profile.SetLanguages(nil)
	profile.SetCategories(nil)
	profile.SetSkills(nil)
	profile.SetTravelCities(nil)

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

// EnsureEmployerProfile creates an empty employer profile if it doesn't exist.
func (s *Service) EnsureEmployerProfile(ctx context.Context, userID uuid.UUID) (*EmployerProfile, error) {
	existing, err := s.employerRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	now := time.Now()
	profile := &EmployerProfile{
		ID:             uuid.New(),
		UserID:         userID,
		CompanyName:    "",
		Rating:         0,
		TotalReviews:   0,
		CastingsPosted: 0,
		IsVerified:     false,
		CreatedAt:      now,
		UpdatedAt:      now,
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
	profile.AvatarURL = s.buildAvatarURL(profile.AvatarFilePath)
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
	profile.AvatarURL = s.buildAvatarURL(profile.AvatarFilePath)
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
	profile.AvatarURL = s.buildAvatarURL(profile.AvatarFilePath)
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
	profile.AvatarURL = s.buildAvatarURL(profile.AvatarFilePath)
	return profile, nil
}

// UpdateModelProfile updates model profile
func (s *Service) UpdateModelProfile(ctx context.Context, userID uuid.UUID, req *UpdateModelProfileRequest) (*ModelProfile, error) {
	profile, err := s.modelRepo.GetByUserID(ctx, userID)
	if err != nil || profile == nil {
		return nil, ErrProfileNotFound
	}

	// Update fields
	if req.Name != "" {
		profile.Name = sql.NullString{String: req.Name, Valid: true}
	}
	if req.Bio != "" {
		profile.Bio = sql.NullString{String: req.Bio, Valid: true}
	}
	if req.Description != "" {
		profile.Description = sql.NullString{String: req.Description, Valid: true}
	}
	if req.City != "" {
		profile.City = sql.NullString{String: req.City, Valid: true}
	}
	if req.Country != "" {
		profile.Country = sql.NullString{String: req.Country, Valid: true}
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
	if req.Skills != nil {
		profile.SetSkills(req.Skills)
	}
	if req.TravelCities != nil {
		profile.SetTravelCities(req.TravelCities)
	}
	if req.BarterAccepted != nil {
		profile.BarterAccepted = *req.BarterAccepted
	}
	if req.AcceptRemoteWork != nil {
		profile.AcceptRemoteWork = *req.AcceptRemoteWork
	}
	// New physical characteristics
	if req.HairColor != "" {
		profile.HairColor = sql.NullString{String: req.HairColor, Valid: true}
	}
	if req.EyeColor != "" {
		profile.EyeColor = sql.NullString{String: req.EyeColor, Valid: true}
	}
	if req.Tattoos != "" {
		profile.Tattoos = sql.NullString{String: req.Tattoos, Valid: true}
	}
	if req.ClothingSize != "" {
		profile.ClothingSize = sql.NullString{String: req.ClothingSize, Valid: true}
	}
	if req.ShoeSize != "" {
		profile.ShoeSize = sql.NullString{String: req.ShoeSize, Valid: true}
	}
	// New professional details
	if req.WorkingHours != "" {
		profile.WorkingHours = sql.NullString{String: req.WorkingHours, Valid: true}
	}
	if req.MinBudget != nil {
		profile.MinBudget = sql.NullFloat64{Float64: *req.MinBudget, Valid: true}
	}
	if req.HourlyRate != nil {
		profile.HourlyRate = sql.NullFloat64{Float64: *req.HourlyRate, Valid: true}
	}
	if req.Experience != nil {
		profile.Experience = sql.NullInt32{Int32: int32(*req.Experience), Valid: true}
	}
	if req.SocialLinks != nil {
		profile.SetSocialLinks(req.SocialLinks)
	}
	// New body measurements
	if req.BustCm != nil {
		profile.BustCm = sql.NullInt32{Int32: int32(*req.BustCm), Valid: true}
	}
	if req.WaistCm != nil {
		profile.WaistCm = sql.NullInt32{Int32: int32(*req.WaistCm), Valid: true}
	}
	if req.HipsCm != nil {
		profile.HipsCm = sql.NullInt32{Int32: int32(*req.HipsCm), Valid: true}
	}
	if req.SkinTone != "" {
		profile.SkinTone = sql.NullString{String: req.SkinTone, Valid: true}
	}
	if req.Specializations != nil {
		if b, err := json.Marshal(req.Specializations); err == nil {
			profile.Specializations = json.RawMessage(b)
		}
	}

	if req.AvatarUploadID != nil {
		profile.AvatarUploadID = uuid.NullUUID{UUID: *req.AvatarUploadID, Valid: true}
	}

	profile.UpdatedAt = time.Now()

	if err := s.modelRepo.Update(ctx, profile); err != nil {
		return nil, err
	}
	profile.AvatarURL = s.buildAvatarURL(profile.AvatarFilePath)

	return profile, nil
}

// UpdateEmployerProfile updates employer profile.
func (s *Service) UpdateEmployerProfile(ctx context.Context, userID uuid.UUID, req *UpdateEmployerProfileRequest) (*EmployerProfile, error) {
	profile, err := s.employerRepo.GetByUserID(ctx, userID)
	if err != nil || profile == nil {
		return nil, ErrProfileNotFound
	}

	if req.CompanyName != "" {
		profile.CompanyName = req.CompanyName
	}
	if req.CompanyType != "" {
		profile.CompanyType = sql.NullString{String: req.CompanyType, Valid: true}
	}
	if req.Description != "" {
		profile.Description = sql.NullString{String: req.Description, Valid: true}
	}
	if req.Website != "" {
		profile.Website = sql.NullString{String: req.Website, Valid: true}
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
	if req.SocialLinks != nil {
		profile.SetSocialLinks(req.SocialLinks)
	}

	if req.AvatarUploadID != nil {
		profile.AvatarUploadID = uuid.NullUUID{UUID: *req.AvatarUploadID, Valid: true}
	}

	profile.UpdatedAt = time.Now()

	if err := s.employerRepo.Update(ctx, profile); err != nil {
		return nil, err
	}
	profile.AvatarURL = s.buildAvatarURL(profile.AvatarFilePath)

	return profile, nil
}

// ListModels returns model profiles with filters
func (s *Service) ListModels(ctx context.Context, filter *Filter, pagination *Pagination) ([]*ModelProfile, int, error) {
	profiles, total, err := s.modelRepo.List(ctx, filter, pagination)
	if err == nil {
		for _, p := range profiles {
			p.AvatarURL = s.buildAvatarURL(p.AvatarFilePath)
		}
	}
	return profiles, total, err
}

// ListPromotedModels returns promoted model profiles
func (s *Service) ListPromotedModels(ctx context.Context, city *string, limit int) ([]*ModelProfile, error) {
	profiles, err := s.modelRepo.ListPromoted(ctx, city, limit)
	if err == nil {
		for _, p := range profiles {
			p.AvatarURL = s.buildAvatarURL(p.AvatarFilePath)
		}
	}
	return profiles, err
}

// IncrementModelViewCount increments mo view count
func (s *Service) IncrementModelViewCount(ctx context.Context, id uuid.UUID) error {
	return s.modelRepo.IncrementViewCount(ctx, id)
}

func (s *Service) GetAdminProfileByUserID(ctx context.Context, userID uuid.UUID) (*AdminProfile, error) {
	profile, err := s.adminRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrProfileNotFound
	}
	return profile, nil
}

func (s *Service) UpdateAdminProfileByUserID(ctx context.Context, userID uuid.UUID, req *UpdateAdminProfileRequest) (*AdminProfile, error) {
	profile, err := s.adminRepo.GetByUserID(ctx, userID)
	if err != nil || profile == nil {
		return nil, ErrProfileNotFound
	}
	if req.Name != "" {
		profile.Name = sql.NullString{String: req.Name, Valid: true}
	}
	if req.Role != "" {
		profile.Role = sql.NullString{String: req.Role, Valid: true}
	}
	if req.AvatarURL != "" {
		profile.AvatarURL = sql.NullString{String: req.AvatarURL, Valid: true}
	}
	profile.UpdatedAt = time.Now()
	if err := s.adminRepo.Update(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}
