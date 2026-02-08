package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/pkg/jwt"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RoleKey   contextKey = "role"
)

// Auth returns middleware that validates JWT
func Auth(jwtService *jwt.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "Missing authorization header")
				return
			}

			// Check Bearer prefix
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Unauthorized(w, "Invalid authorization header format")
				return
			}

			// Validate token
			claims, err := jwtService.ValidateAccessToken(parts[1])
			if err != nil {
				if err == jwt.ErrExpiredToken {
					response.Unauthorized(w, "Token expired")
				} else {
					response.Unauthorized(w, "Invalid token")
				}
				return
			}

			// Task 2: BAN ENFORCEMENT - Check if user is banned
			if claims.IsBanned {
				response.Forbidden(w, "Your account has been banned")
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetRole extracts role from context
func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(RoleKey).(string); ok {
		return role
	}
	return ""
}

// RequireRole returns middleware that checks user role
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole := GetRole(r.Context())

			for _, role := range roles {
				if userRole == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Forbidden(w, "Insufficient permissions")
		})
	}
}

// RequireModel returns middleware that requires model role
func RequireModel() func(http.Handler) http.Handler {
	return RequireRole("model")
}

// RequireEmployer returns middleware that requires employer or agency role
func RequireEmployer() func(http.Handler) http.Handler {
	return RequireRole("employer", "agency")
}

// RequireAdmin returns middleware that requires admin role
func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole("admin")
}
