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

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims represents access JWT claims
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	IsBanned bool      `json:"is_banned"`
	Type     string    `json:"type"`
	jwt.RegisteredClaims
}

// RefreshClaims represents refresh JWT claims
type RefreshClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Type   string    `json:"type"`
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
	return &Service{secret: []byte(secret), accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// GenerateAccessToken generates access token
func (s *Service) GenerateAccessToken(userID uuid.UUID, role string, isBanned bool) (string, error) {
	claims := Claims{
		UserID:   userID,
		Role:     role,
		IsBanned: isBanned,
		Type:     TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GenerateRefreshToken generates signed refresh JWT
func (s *Service) GenerateRefreshToken(userID uuid.UUID) (token string, jti string, expiresAt time.Time, err error) {
	now := time.Now()
	jti = uuid.New().String()
	expiresAt = now.Add(s.refreshTTL)
	claims := RefreshClaims{
		UserID: userID,
		Type:   TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = jwtToken.SignedString(s.secret)
	return
}

// HashRefreshToken hashes refresh token for storage
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
	if !ok || !token.Valid || claims.Type != TokenTypeAccess {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ValidateRefreshToken validates and parses refresh token
func (s *Service) ValidateRefreshToken(tokenString string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
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
	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid || claims.Type != TokenTypeRefresh {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// GenerateOpaqueToken kept for compatibility in other domains
func GenerateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) GetAccessTTL() time.Duration  { return s.accessTTL }
func (s *Service) GetRefreshTTL() time.Duration { return s.refreshTTL }
