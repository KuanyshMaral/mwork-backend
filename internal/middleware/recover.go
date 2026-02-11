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
				stack := string(debug.Stack())

				if panicRecorder, ok := w.(interface {
					SetPanicDetails(err interface{}, stack string)
				}); ok {
					panicRecorder.SetPanicDetails(err, stack)
				}

				// Log the panic with stack trace
				log.Error().
					Interface("error", err).
					Str("stack", string(debug.Stack())).
					Str("stack", stack).
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Str("request_id", r.Header.Get("X-Request-ID")).
					Msg("Panic recovered")

				// Return 500 error to client
				response.InternalError(w)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
