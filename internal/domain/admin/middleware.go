package admin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/pkg/response"
)

// AdminClaims for admin JWT tokens
type AdminClaims struct {
	AdminID uuid.UUID `json:"admin_id"`
	Email   string    `json:"email"`
	Role    Role      `json:"role"`
	jwt.RegisteredClaims
}

// AdminContextKey for context values
type AdminContextKey string

const (
	ContextAdminID   AdminContextKey = "admin_id"
	ContextAdminRole AdminContextKey = "admin_role"
)

// JWTService for generating admin tokens
type JWTService struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTService creates admin JWT service
func NewJWTService(secret string, ttl time.Duration) *JWTService {
	return &JWTService{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// GenerateToken creates a new admin JWT
func (s *JWTService) GenerateToken(admin *AdminUser) (string, error) {
	claims := AdminClaims{
		AdminID: admin.ID,
		Email:   admin.Email,
		Role:    admin.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   admin.ID.String(),
			Issuer:    "mwork-admin",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken validates admin JWT and returns claims
func (s *JWTService) ValidateToken(tokenString string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AdminClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// AuthMiddleware creates admin authentication middleware
func AuthMiddleware(jwtSvc *JWTService, adminSvc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "Missing authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Unauthorized(w, "Invalid authorization header format")
				return
			}

			// Validate token
			claims, err := jwtSvc.ValidateToken(parts[1])
			if err != nil {
				response.Unauthorized(w, "Invalid or expired token")
				return
			}

			// Check admin still exists and is active
			admin, err := adminSvc.GetAdminByID(r.Context(), claims.AdminID)
			if err != nil || admin == nil {
				response.Unauthorized(w, "Admin not found")
				return
			}

			if !admin.IsActive {
				response.Forbidden(w, "Admin account is inactive")
				return
			}

			// Add admin info to context
			ctx := context.WithValue(r.Context(), ContextAdminID, claims.AdminID)
			ctx = context.WithValue(ctx, ContextAdminRole, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission middleware checks for specific permission
func RequirePermission(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(ContextAdminRole).(Role)
			if !ok {
				response.Forbidden(w, "Permission denied")
				return
			}

			// Check permission
			permissions, exists := RolePermissions[role]
			if !exists {
				response.Forbidden(w, "Permission denied")
				return
			}

			hasPermission := false
			for _, p := range permissions {
				if p == perm {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				response.Forbidden(w, "Permission denied")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAdminID extracts admin ID from context
func GetAdminID(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(ContextAdminID).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// GetAdminRole extracts admin role from context
func GetAdminRole(ctx context.Context) Role {
	role, ok := ctx.Value(ContextAdminRole).(Role)
	if !ok {
		return ""
	}
	return role
}
