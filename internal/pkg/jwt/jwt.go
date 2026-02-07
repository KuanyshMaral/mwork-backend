// internal/pkg/jwt/jwt.go
package jwt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims represents JWT claims
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	IsBanned bool      `json:"is_banned"` // Task 2: Add banned status to token
	jwt.RegisteredClaims
}

// Service handles JWT operations
type Service struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewService creates JWT service
func NewService(secret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// GenerateAccessToken generates access token
func (s *Service) GenerateAccessToken(userID uuid.UUID, role string, isBanned bool) (string, error) {
	claims := Claims{
		UserID:   userID,
		Role:     role,
		IsBanned: isBanned,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(), // jti
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GenerateRefreshToken generates refresh token (32 bytes -> 64 hex chars)
func (s *Service) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32) // 32 bytes = 256-bit
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashRefreshToken hashes refresh token for storage (so raw token isn't stored)
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ValidateAccessToken validates and parses access token
func (s *Service) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetAccessTTL returns access token TTL
func (s *Service) GetAccessTTL() time.Duration {
	return s.accessTTL
}

// GetRefreshTTL returns refresh token TTL
func (s *Service) GetRefreshTTL() time.Duration {
	return s.refreshTTL
}
