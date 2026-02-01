// internal/domain/auth/service.go
package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/mwork/mwork-api/internal/domain/organization"
	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/pkg/jwt"
	"github.com/mwork/mwork-api/internal/pkg/password"
)

// Service handles authentication business logic
type Service struct {
	userRepo         user.Repository
	jwtService       *jwt.Service
	redis            *redis.Client // nil if Redis disabled
	employerProfRepo EmployerProfileRepository
	orgRepo          OrganizationRepository
}

// EmployerProfileRepository defines employer profile operations needed by auth
type EmployerProfileRepository interface {
	Create(ctx context.Context, profile *EmployerProfile) error
}

// OrganizationRepository defines organization operations needed by auth
type OrganizationRepository interface {
	Create(ctx context.Context, org *organization.Organization) error
	GetByBIN(ctx context.Context, bin string) (*organization.Organization, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// EmployerProfile represents an employer profile entity
type EmployerProfile struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	CompanyName   string
	Description   string
	Website       string
	ContactPerson string
	ContactPhone  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewService creates auth service
func NewService(userRepo user.Repository, jwtService *jwt.Service, redis *redis.Client, employerProfRepo EmployerProfileRepository, orgRepo OrganizationRepository) *Service {
	return &Service{
		userRepo:         userRepo,
		jwtService:       jwtService,
		redis:            redis,
		employerProfRepo: employerProfRepo,
		orgRepo:          orgRepo,
	}
}

// Register creates new user account
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// 1. Check if email exists
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// 2. Validate role
	if !user.IsValidRole(req.Role) {
		return nil, ErrInvalidRole
	}

	var org *organization.Organization
	if req.Role == string(user.RoleEmployer) {
		if s.orgRepo == nil {
			return nil, fmt.Errorf("organization repository is not configured")
		}
		existingOrg, _ := s.orgRepo.GetByBIN(ctx, req.BinIIN)
		if existingOrg != nil {
			return nil, organization.ErrBINAlreadyExists
		}
	}

	// 3. Hash password
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	// 4. Create user
	now := time.Now()
	verificationStatus := user.VerificationNone
	var submittedAt sql.NullTime
	if req.Role == string(user.RoleEmployer) {
		verificationStatus = user.VerificationPending
		submittedAt = sql.NullTime{Time: now, Valid: true}
	}
	u := &user.User{
		ID:                      uuid.New(),
		Email:                   req.Email,
		PasswordHash:            hash,
		Role:                    user.Role(req.Role),
		EmailVerified:           false,
		UserVerificationStatus:  verificationStatus,
		VerificationSubmittedAt: submittedAt,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}

	if req.Role == string(user.RoleEmployer) {
		org = &organization.Organization{
			ID:                 uuid.New(),
			LegalName:          req.CompanyName,
			BrandName:          sql.NullString{String: req.CompanyName, Valid: true},
			BinIIN:             req.BinIIN,
			OrgType:            organization.OrgTypeTOO,
			Phone:              sql.NullString{String: req.ContactPhone, Valid: true},
			Email:              sql.NullString{String: req.ContactEmail, Valid: true},
			ContactPerson:      sql.NullString{String: req.ContactName, Valid: true},
			ContactPhone:       sql.NullString{String: req.ContactPhone, Valid: true},
			ContactTelegram:    sql.NullString{String: req.ContactTelegram, Valid: req.ContactTelegram != ""},
			ContactWhatsApp:    sql.NullString{String: req.ContactWhatsApp, Valid: req.ContactWhatsApp != ""},
			VerificationStatus: organization.VerificationPending,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		if err := s.orgRepo.Create(ctx, org); err != nil {
			_ = s.userRepo.Delete(ctx, u.ID)
			return nil, err
		}

		u.OrganizationID = uuid.NullUUID{UUID: org.ID, Valid: true}
		if err := s.userRepo.Update(ctx, u); err != nil {
			_ = s.userRepo.Delete(ctx, u.ID)
			_ = s.orgRepo.Delete(ctx, org.ID)
			return nil, err
		}

		if s.employerProfRepo != nil {
			profile := &EmployerProfile{
				ID:            uuid.New(),
				UserID:        u.ID,
				CompanyName:   req.CompanyName,
				ContactPerson: req.ContactName,
				ContactPhone:  req.ContactPhone,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if err := s.employerProfRepo.Create(ctx, profile); err != nil {
				_ = s.userRepo.Delete(ctx, u.ID)
				_ = s.orgRepo.Delete(ctx, org.ID)
				return nil, err
			}
		}
	}

	// 5. Generate tokens
	return s.generateTokens(ctx, u)
}

// RegisterAgency creates new agency user account with employer profile
func (s *Service) RegisterAgency(ctx context.Context, req *AgencyRegisterRequest) (*AuthResponse, error) {
	// 1. Check if email exists
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// 2. Hash password
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	// 3. Create user with role='agency'
	now := time.Now()
	u := &user.User{
		ID:            uuid.New(),
		Email:         req.Email,
		PasswordHash:  hash,
		Role:          "agency", // Agency role
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}

	// 4. Create employer profile for agency
	bio := fmt.Sprintf("Managed by: %s", req.ContactPerson)
	profile := &EmployerProfile{
		ID:            uuid.New(),
		UserID:        u.ID,
		CompanyName:   req.CompanyName,
		Website:       req.Website,
		Description:   bio,
		ContactPerson: req.ContactPerson,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.employerProfRepo.Create(ctx, profile); err != nil {
		// Rollback: delete user if profile creation fails
		_ = s.userRepo.Delete(ctx, u.ID)
		return nil, err
	}

	// 5. Generate tokens
	return s.generateTokens(ctx, u)
}

// Login authenticates user
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	// 1. Find user
	u, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || u == nil {
		return nil, ErrInvalidCredentials
	}

	// 2. Verify password
	if !password.Verify(req.Password, u.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Check if banned
	if u.IsBanned {
		return nil, ErrUserBanned
	}

	// 3. Generate tokens
	return s.generateTokens(ctx, u)
}

// Refresh refreshes access token using refresh token
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	if refreshToken == "" {
		return nil, ErrRefreshTokenRequired
	}

	// 1. Validate refresh token in Redis (we store hash(refresh))
	refreshHash := jwt.HashRefreshToken(refreshToken)
	userID, err := s.getRefreshToken(ctx, refreshHash)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// 2. Get user
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}

	// 3. Delete old refresh token (token rotation)
	_ = s.deleteRefreshToken(ctx, refreshHash)

	// 4. Generate new tokens
	return s.generateTokens(ctx, u)
}

// Logout invalidates refresh token
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil // Nothing to logout
	}

	// delete by hash(refresh)
	refreshHash := jwt.HashRefreshToken(refreshToken)
	return s.deleteRefreshToken(ctx, refreshHash)
}

// GetCurrentUser returns current user by ID
func (s *Service) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}

	resp := NewUserResponse(u.ID, u.Email, string(u.Role), u.EmailVerified, string(u.UserVerificationStatus), u.CreatedAt)
	return &resp, nil
}

// generateTokens creates access and refresh tokens
func (s *Service) generateTokens(ctx context.Context, u *user.User) (*AuthResponse, error) {
	// Generate access token
	accessToken, err := s.jwtService.GenerateAccessToken(u.ID, string(u.Role))
	if err != nil {
		return nil, err
	}

	// Generate refresh token (32 bytes hex)
	refreshToken, err := s.jwtService.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Store hash(refresh) in Redis
	refreshHash := jwt.HashRefreshToken(refreshToken)
	if err := s.storeRefreshToken(ctx, refreshHash, u.ID); err != nil {
		return nil, err
	}

	return &AuthResponse{
		User: NewUserResponse(u.ID, u.Email, string(u.Role), u.EmailVerified, string(u.UserVerificationStatus), u.CreatedAt),
		Tokens: TokensResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken, // return raw refresh to client
			ExpiresIn:    int(s.jwtService.GetAccessTTL().Seconds()),
		},
	}, nil
}

// Redis helpers (handle nil redis gracefully)
func (s *Service) storeRefreshToken(ctx context.Context, token string, userID uuid.UUID) error {
	if s.redis == nil {
		return nil // Skip if Redis not configured
	}
	return s.redis.Set(ctx, "refresh:"+token, userID.String(), s.jwtService.GetRefreshTTL()).Err()
}

func (s *Service) getRefreshToken(ctx context.Context, token string) (uuid.UUID, error) {
	if s.redis == nil {
		// Without Redis, refresh tokens don't work
		return uuid.Nil, ErrInvalidRefreshToken
	}
	val, err := s.redis.Get(ctx, "refresh:"+token).Result()
	if err != nil {
		return uuid.Nil, ErrInvalidRefreshToken
	}
	return uuid.Parse(val)
}

func (s *Service) deleteRefreshToken(ctx context.Context, token string) error {
	if s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, "refresh:"+token).Err()
}

// FindByEmail finds user by email address
func (s *Service) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// MarkEmailVerified marks user's email as verified
func (s *Service) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	return s.userRepo.UpdateEmailVerified(ctx, userID, true)
}

// UpdatePassword updates user's password
func (s *Service) UpdatePassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(ctx, userID, hash)
}
