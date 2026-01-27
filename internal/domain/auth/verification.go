package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/mwork/mwork-api/internal/pkg/email"
)

// VerificationService handles email verification and password reset
type VerificationService struct {
	redis        *redis.Client
	emailService *email.Service
	baseURL      string
}

// NewVerificationService creates verification service
func NewVerificationService(redis *redis.Client, emailService *email.Service, baseURL string) *VerificationService {
	return &VerificationService{
		redis:        redis,
		emailService: emailService,
		baseURL:      baseURL,
	}
}

// Verification code settings
const (
	VerificationCodeLength = 6
	VerificationCodeTTL    = 15 * time.Minute
	ResetTokenTTL          = 1 * time.Hour
)

// Redis key prefixes
const (
	keyPrefixVerification = "verify:"
	keyPrefixReset        = "reset:"
)

// GenerateVerificationCode generates a 6-digit code and stores in Redis
func (s *VerificationService) GenerateVerificationCode(ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate 6-digit code
	code := generateNumericCode(VerificationCodeLength)

	// Store in Redis with TTL
	key := keyPrefixVerification + userID.String()
	if err := s.redis.Set(ctx, key, code, VerificationCodeTTL).Err(); err != nil {
		return "", fmt.Errorf("failed to store verification code: %w", err)
	}

	return code, nil
}

// SendVerificationEmail generates code and sends verification email
func (s *VerificationService) SendVerificationEmail(ctx context.Context, userID uuid.UUID, email, name string) error {
	code, err := s.GenerateVerificationCode(ctx, userID)
	if err != nil {
		return err
	}

	// Send email via email service
	s.emailService.Queue(email, name, "verification", "Подтвердите ваш email", map[string]string{
		"UserName": name,
		"Code":     code,
	})

	return nil
}

// VerifyEmail checks the code and returns true if valid
func (s *VerificationService) VerifyEmail(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	key := keyPrefixVerification + userID.String()

	storedCode, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil // Code expired or not found
	}
	if err != nil {
		return false, err
	}

	if storedCode != code {
		return false, nil // Invalid code
	}

	// Delete code after successful verification
	s.redis.Del(ctx, key)

	return true, nil
}

// GenerateResetToken generates a password reset token
func (s *VerificationService) GenerateResetToken(ctx context.Context, userID uuid.UUID) (string, error) {
	// Generate secure random token
	token := generateSecureToken(32)

	// Store token -> userID mapping
	key := keyPrefixReset + token
	if err := s.redis.Set(ctx, key, userID.String(), ResetTokenTTL).Err(); err != nil {
		return "", fmt.Errorf("failed to store reset token: %w", err)
	}

	return token, nil
}

// SendPasswordResetEmail generates token and sends reset email
func (s *VerificationService) SendPasswordResetEmail(ctx context.Context, userID uuid.UUID, email, name string) error {
	token, err := s.GenerateResetToken(ctx, userID)
	if err != nil {
		return err
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)

	// Send email
	s.emailService.Queue(email, name, "password_reset", "Сброс пароля", map[string]string{
		"UserName": name,
		"ResetURL": resetURL,
	})

	return nil
}

// ValidateResetToken checks if token is valid and returns userID
func (s *VerificationService) ValidateResetToken(ctx context.Context, token string) (uuid.UUID, error) {
	key := keyPrefixReset + token

	userIDStr, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return uuid.Nil, fmt.Errorf("invalid or expired reset token")
	}
	if err != nil {
		return uuid.Nil, err
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

// InvalidateResetToken removes a reset token after use
func (s *VerificationService) InvalidateResetToken(ctx context.Context, token string) {
	key := keyPrefixReset + token
	s.redis.Del(ctx, key)
}

// Helpers

func generateNumericCode(length int) string {
	const digits = "0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b)
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}
