// internal/domain/auth/service.go
package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/jwt"
	"github.com/mwork/mwork-api/internal/pkg/password"
	"github.com/mwork/mwork-api/internal/pkg/photostudio"
)

// Service handles authentication business logic
type Service struct {
	userRepo                 user.Repository
	modelProfRepo            ModelProfileRepository
	jwtService               *jwt.Service
	refreshTokenRepo         RefreshTokenStore
	employerProfRepo         EmployerProfileRepository
	photoStudioClient        PhotoStudioClient
	photoStudioSyncEnabled   bool
	photoStudioTimeout       time.Duration
	verificationCodeRepo     *VerificationCodeRepository
	verificationCodePepper   string
	verificationCodeLogInDev bool
}

type RefreshTokenStore interface {
	Create(ctx context.Context, rec *RefreshTokenRecord) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshTokenRecord, error)
	MarkUsed(ctx context.Context, tokenHash string) error
	RevokeByTokenHash(ctx context.Context, tokenHash string) error
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error
}

// ModelProfileRepository defines model profile operations needed by auth
type ModelProfileRepository interface {
	Create(ctx context.Context, profile *ModelProfile) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*ModelProfile, error)
}

// EmployerProfileRepository defines employer profile operations needed by auth
type EmployerProfileRepository interface {
	Create(ctx context.Context, profile *EmployerProfile) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*EmployerProfile, error)
}

// ModelProfile represents a model profile entity
type ModelProfile struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// EmployerProfile represents an employer profile entity
type EmployerProfile struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	CompanyName   string
	Description   string
	Website       string
	ContactPerson string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PhotoStudioClient defines PhotoStudio sync client interface.
type PhotoStudioClient interface {
	SyncUser(ctx context.Context, payload photostudio.SyncUserPayload) error
}

// NewService creates auth service
func NewService(
	userRepo user.Repository,
	modelProfRepo ModelProfileRepository,
	jwtService *jwt.Service,
	refreshTokenRepo RefreshTokenStore,
	employerProfRepo EmployerProfileRepository,
	photoStudioClient PhotoStudioClient,
	photoStudioSyncEnabled bool,
	photoStudioTimeout time.Duration,
	verificationCodeRepo *VerificationCodeRepository,
	verificationCodePepper string,
	verificationCodeLogInDev bool,
) *Service {
	if photoStudioTimeout <= 0 {
		photoStudioTimeout = 10 * time.Second
	}
	return &Service{
		userRepo:                 userRepo,
		modelProfRepo:            modelProfRepo,
		jwtService:               jwtService,
		refreshTokenRepo:         refreshTokenRepo,
		employerProfRepo:         employerProfRepo,
		photoStudioClient:        photoStudioClient,
		photoStudioSyncEnabled:   photoStudioSyncEnabled,
		photoStudioTimeout:       photoStudioTimeout,
		verificationCodeRepo:     verificationCodeRepo,
		verificationCodePepper:   verificationCodePepper,
		verificationCodeLogInDev: verificationCodeLogInDev,
	}
}

// Register creates new user account
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	requestID := middleware.GetRequestID(ctx)
	req.Email = normalizeEmail(req.Email)
	// 1. Check if email exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		wrappedErr := wrapRegisterError("check-user-by-email", err)
		log.Error().Err(wrappedErr).Str("request_id", requestID).Str("email", req.Email).Msg("failed to check existing user by email")
		return nil, wrappedErr
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// 2. Validate role
	if !user.IsValidRole(req.Role) {
		log.Warn().Str("email", req.Email).Str("role", req.Role).Msg("invalid role in register request")
		return nil, ErrInvalidRole
	}

	// 3. Hash password
	hash, err := password.Hash(req.Password)
	if err != nil {
		wrappedErr := wrapRegisterError("hash-password", err)
		log.Error().Err(wrappedErr).Str("request_id", requestID).Str("email", req.Email).Msg("failed to hash password during register")
		return nil, wrappedErr
	}

	// 4. Create user
	now := time.Now()
	u := &user.User{
		ID:            uuid.New(),
		Email:         req.Email,
		PasswordHash:  hash,
		Role:          user.Role(req.Role),
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		wrappedErr := wrapRegisterError("create-user", err)
		e := log.Error().Err(wrappedErr).Str("request_id", requestID).Str("email", req.Email).Str("user_id", u.ID.String())
		if details := extractDBErrorDetails(err); details != nil {
			e.Str("db_sqlstate", details.SQLState).Str("db_constraint", details.Constraint).Str("db_table", details.Table).Str("db_column", details.Column).Str("db_detail", details.Detail).Str("db_message", details.Message)
		}
		e.Msg("failed to create user")
		if isEmailAlreadyExistsError(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, wrappedErr
	}

	s.syncPhotoStudioUser(photostudio.SyncUserPayload{
		MWorkUserID: u.ID.String(),
		Email:       u.Email,
		Role:        string(u.Role),
	})

	// 5. Generate tokens
	return s.generateTokens(ctx, u)
}

// RegisterAgency creates new agency user account with employer profile
func (s *Service) RegisterAgency(ctx context.Context, req *AgencyRegisterRequest) (*AuthResponse, error) {
	requestID := middleware.GetRequestID(ctx)
	req.Email = normalizeEmail(req.Email)
	// 1. Check if email exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		wrappedErr := wrapRegisterError("check-agency-by-email", err)
		log.Error().Err(wrappedErr).Str("request_id", requestID).Str("email", req.Email).Msg("failed to check existing agency by email")
		return nil, wrappedErr
	}
	if existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// 2. Hash password
	hash, err := password.Hash(req.Password)
	if err != nil {
		wrappedErr := wrapRegisterError("agency-hash-password", err)
		log.Error().Err(wrappedErr).Str("request_id", requestID).Str("email", req.Email).Msg("failed to hash password during agency register")
		return nil, wrappedErr
	}

	// 3. Create user with role='agency'
	now := time.Now()
	u := &user.User{
		ID:            uuid.New(),
		Email:         req.Email,
		PasswordHash:  hash,
		Role:          user.RoleAgency, // Agency role
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.userRepo.Create(ctx, u); err != nil {
		wrappedErr := wrapRegisterError("create-agency-user", err)
		e := log.Error().Err(wrappedErr).Str("request_id", requestID).Str("email", req.Email).Str("user_id", u.ID.String())
		if details := extractDBErrorDetails(err); details != nil {
			e.Str("db_sqlstate", details.SQLState).Str("db_constraint", details.Constraint).Str("db_table", details.Table).Str("db_column", details.Column).Str("db_detail", details.Detail).Str("db_message", details.Message)
		}
		e.Msg("failed to create agency user")
		if isEmailAlreadyExistsError(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, wrappedErr
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
		wrappedErr := wrapRegisterError("create-agency-profile", err)
		e := log.Error().
			Err(wrappedErr).
			Str("request_id", requestID).
			Str("email", req.Email).
			Str("user_id", u.ID.String()).
			Str("profile_id", profile.ID.String())
		if details := extractDBErrorDetails(err); details != nil {
			e.Str("db_sqlstate", details.SQLState).Str("db_constraint", details.Constraint).Str("db_table", details.Table).Str("db_column", details.Column).Str("db_detail", details.Detail).Str("db_message", details.Message)
		}
		e.Msg("failed to create employer profile for agency")
		// Rollback: delete user if profile creation fails
		_ = s.userRepo.Delete(ctx, u.ID)
		return nil, wrappedErr
	}

	s.syncPhotoStudioUser(photostudio.SyncUserPayload{
		MWorkUserID: u.ID.String(),
		Email:       u.Email,
		Role:        string(u.Role),
	})

	// 5. Generate tokens
	return s.generateTokens(ctx, u)
}

// Login authenticates user
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	log.Info().Str("email", req.Email).Msg("Login attempt")

	// 1. Find user
	u, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("Database error during user lookup")
		return nil, ErrInvalidCredentials
	}
	if u == nil {
		log.Warn().Str("email", req.Email).Msg("User not found")
		return nil, ErrInvalidCredentials
	}

	log.Info().Str("user_id", u.ID.String()).Str("email", u.Email).Msg("User found")

	// 2. Verify password
	passwordValid := password.Verify(req.Password, u.PasswordHash)
	if !passwordValid {
		log.Warn().Str("email", req.Email).Msg("Password verification failed")
		return nil, ErrInvalidCredentials
	}

	// Check if banned
	if u.IsBanned {
		log.Warn().Str("email", req.Email).Msg("User is banned")
		return nil, ErrUserBanned
	}

	if err := s.ensureProfileExists(ctx, u); err != nil {
		log.Error().Err(err).Str("user_id", u.ID.String()).Str("role", string(u.Role)).Msg("failed to ensure profile exists on login")
		return nil, err
	}

	log.Info().Str("email", req.Email).Msg("Login successful")
	// 3. Generate tokens
	return s.generateTokens(ctx, u)
}

func (s *Service) ensureProfileExists(ctx context.Context, u *user.User) error {
	now := time.Now()

	if u.IsModel() {
		existing, err := s.modelProfRepo.GetByUserID(ctx, u.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			return nil
		}

		return s.modelProfRepo.Create(ctx, &ModelProfile{
			ID:        uuid.New(),
			UserID:    u.ID,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if u.IsEmployer() || u.IsAgency() {
		existing, err := s.employerProfRepo.GetByUserID(ctx, u.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			return nil
		}

		return s.employerProfRepo.Create(ctx, &EmployerProfile{
			ID:          uuid.New(),
			UserID:      u.ID,
			CompanyName: "",
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	return nil
}

func (s *Service) RequestVerificationCode(ctx context.Context, userID uuid.UUID) (string, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return "", ErrUserNotFound
	}

	if u.EmailVerified {
		return "already_verified", nil
	}

	if s.verificationCodeRepo == nil {
		return "", fmt.Errorf("verification code repository is not configured")
	}

	code := generateNumericCode(VerificationCodeLength)
	hash := s.hashVerificationCode(code)
	expiresAt := time.Now().Add(verificationCodeTTL)
	if err := s.verificationCodeRepo.Upsert(ctx, userID, hash, expiresAt); err != nil {
		return "", err
	}

	if s.verificationCodeLogInDev {
		log.Info().Str("user_id", userID.String()).Str("verification_code", code).Msg("DEV verification code generated")
	}

	return "sent", nil
}

func (s *Service) ConfirmVerificationCode(ctx context.Context, userID uuid.UUID, code string) (string, error) {
	if s.verificationCodeRepo == nil {
		return "", fmt.Errorf("verification code repository is not configured")
	}

	rec, err := s.verificationCodeRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrInvalidVerificationCode
		}
		return "", err
	}

	now := time.Now()
	if rec.UsedAt != nil || now.After(rec.ExpiresAt) || rec.Attempts >= verificationCodeMaxAttempts {
		_ = s.verificationCodeRepo.Invalidate(ctx, userID)
		return "", ErrInvalidVerificationCode
	}

	if rec.CodeHash != s.hashVerificationCode(code) {
		attempts, incErr := s.verificationCodeRepo.IncrementAttempts(ctx, userID)
		if incErr != nil {
			return "", incErr
		}
		if attempts >= verificationCodeMaxAttempts {
			_ = s.verificationCodeRepo.Invalidate(ctx, userID)
		}
		return "", ErrInvalidVerificationCode
	}

	if err := s.userRepo.UpdateEmailVerified(ctx, userID, true); err != nil {
		return "", err
	}
	if err := s.verificationCodeRepo.MarkUsed(ctx, userID); err != nil {
		return "", err
	}

	return "verified", nil
}

func (s *Service) hashVerificationCode(code string) string {
	sum := sha256.Sum256([]byte(code + s.verificationCodePepper))
	return fmt.Sprintf("%x", sum[:])
}

// Refresh refreshes access token using refresh token
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	if refreshToken == "" {
		return nil, ErrRefreshTokenRequired
	}
	if s.refreshTokenRepo == nil {
		return nil, ErrInvalidRefreshToken
	}

	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	refreshHash := jwt.HashRefreshToken(refreshToken)
	rec, err := s.refreshTokenRepo.GetByTokenHash(ctx, refreshHash)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	now := time.Now()
	if rec.UserID != claims.UserID || rec.JTI != claims.ID || rec.RevokedAt.Valid || rec.UsedAt.Valid || now.After(rec.ExpiresAt) {
		_ = s.refreshTokenRepo.RevokeAllByUserID(ctx, rec.UserID)
		return nil, ErrInvalidRefreshToken
	}

	if err := s.refreshTokenRepo.MarkUsed(ctx, refreshHash); err != nil {
		return nil, ErrInvalidRefreshToken
	}

	u, err := s.userRepo.GetByID(ctx, rec.UserID)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}

	return s.generateTokens(ctx, u)
}

// Logout invalidates refresh token
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" || s.refreshTokenRepo == nil {
		return nil
	}
	refreshHash := jwt.HashRefreshToken(refreshToken)
	return s.refreshTokenRepo.RevokeByTokenHash(ctx, refreshHash)
}

// GetCurrentUser returns current user by ID
func (s *Service) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}

	resp := NewUserResponse(u.ID, u.Email, string(u.Role), u.EmailVerified, u.CreatedAt)
	return &resp, nil
}

// generateTokens creates access and refresh tokens
func (s *Service) generateTokens(ctx context.Context, u *user.User) (*AuthResponse, error) {
	// Generate access token with banned status
	accessToken, err := s.jwtService.GenerateAccessToken(u.ID, string(u.Role), u.IsBanned)
	if err != nil {
		return nil, err
	}

	// Generate signed refresh token
	refreshToken, refreshJTI, refreshExpiresAt, err := s.jwtService.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, err
	}

	if s.refreshTokenRepo == nil {
		return nil, ErrInvalidRefreshToken
	}

	refreshHash := jwt.HashRefreshToken(refreshToken)
	if err := s.refreshTokenRepo.Create(ctx, &RefreshTokenRecord{
		ID:        uuid.New(),
		UserID:    u.ID,
		TokenHash: refreshHash,
		JTI:       refreshJTI,
		ExpiresAt: refreshExpiresAt,
	}); err != nil {
		wrappedErr := wrapRegisterError("create-refresh-token", err)
		e := log.Error().Err(wrappedErr).Str("request_id", middleware.GetRequestID(ctx)).Str("user_id", u.ID.String())
		if details := extractDBErrorDetails(err); details != nil {
			e.Str("db_sqlstate", details.SQLState).Str("db_constraint", details.Constraint).Str("db_table", details.Table).Str("db_column", details.Column).Str("db_detail", details.Detail).Str("db_message", details.Message)
		}
		e.Msg("failed to persist refresh token")
		return nil, wrappedErr
	}

	return &AuthResponse{
		User: NewUserResponse(u.ID, u.Email, string(u.Role), u.EmailVerified, u.CreatedAt),
		Tokens: TokensResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken, // return raw refresh to client
			ExpiresIn:    int(s.jwtService.GetAccessTTL().Seconds()),
			TokenType:    "Bearer",
		},
	}, nil
}

func (s *Service) syncPhotoStudioUser(payload photostudio.SyncUserPayload) {
	if !s.photoStudioSyncEnabled || s.photoStudioClient == nil || payload.MWorkUserID == "" {
		return
	}

	go func(p photostudio.SyncUserPayload) {
		ctx, cancel := context.WithTimeout(context.Background(), s.photoStudioTimeout)
		defer cancel()

		start := time.Now()
		err := s.photoStudioClient.SyncUser(ctx, p)
		duration := time.Since(start)

		if err != nil {
			log.Warn().
				Err(err).
				Str("user_id", p.MWorkUserID).
				Str("email", p.Email).
				Str("role", p.Role).
				Dur("duration", duration).
				Msg("photostudio sync failed")
			return
		}

		log.Info().
			Str("user_id", p.MWorkUserID).
			Str("email", p.Email).
			Str("role", p.Role).
			Dur("duration", duration).
			Msg("photostudio sync ok")
	}(payload)
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
