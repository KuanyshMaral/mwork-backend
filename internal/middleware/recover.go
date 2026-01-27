package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/rs/zerolog/log"
)

// Recover is a middleware that recovers from panics
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace
				log.Error().
					Interface("error", err).
					Str("stack", string(debug.Stack())).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Msg("Panic recovered")

				// Return 500 error to client
				response.InternalError(w)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
