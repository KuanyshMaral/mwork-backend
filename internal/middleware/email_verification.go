package middleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// RequireVerifiedEmail blocks access for authenticated users with unverified email,
// except for whitelisted paths.
func RequireVerifiedEmail(userRepo user.Repository, whitelist []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			for _, allowed := range whitelist {
				if path == allowed || strings.HasPrefix(path, allowed) {
					next.ServeHTTP(w, r)
					return
				}
			}

			userID := GetUserID(r.Context())
			if userID == uuid.Nil {
				response.Unauthorized(w, "Authentication required")
				return
			}

			u, err := userRepo.GetByID(r.Context(), userID)
			if err != nil || u == nil {
				response.Unauthorized(w, "Authentication required")
				return
			}

			if !u.EmailVerified {
				response.Error(w, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Email is not verified")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
