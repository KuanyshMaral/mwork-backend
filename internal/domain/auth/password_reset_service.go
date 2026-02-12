package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/mwork/mwork-api/internal/pkg/email"
)

const (
	ResetTokenTTL = 1 * time.Hour
)

const keyPrefixReset = "reset:"

// PasswordResetService handles password reset tokens and email notifications.
type PasswordResetService struct {
	redis        *redis.Client
	emailService *email.Service
	baseURL      string
}

func NewPasswordResetService(redis *redis.Client, emailService *email.Service, baseURL string) *PasswordResetService {
	return &PasswordResetService{
		redis:        redis,
		emailService: emailService,
		baseURL:      baseURL,
	}
}

func (s *PasswordResetService) GenerateResetToken(ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return "", err
	}

	key := keyPrefixReset + token
	if err := s.redis.Set(ctx, key, userID.String(), ResetTokenTTL).Err(); err != nil {
		return "", fmt.Errorf("failed to store reset token: %w", err)
	}

	return token, nil
}

func (s *PasswordResetService) SendPasswordResetEmail(ctx context.Context, userID uuid.UUID, emailAddr, name string) error {
	token, err := s.GenerateResetToken(ctx, userID)
	if err != nil {
		return err
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)
	s.emailService.Queue(emailAddr, name, "password_reset", "Сброс пароля", map[string]string{
		"UserName": name,
		"ResetURL": resetURL,
	})

	return nil
}

func (s *PasswordResetService) ValidateResetToken(ctx context.Context, token string) (uuid.UUID, error) {
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

func (s *PasswordResetService) InvalidateResetToken(ctx context.Context, token string) {
	s.redis.Del(ctx, keyPrefixReset+token)
}
