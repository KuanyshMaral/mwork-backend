package middleware

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger is a middleware that logs HTTP requests
const maxLoggedErrorBody = 2048

// Logger is a middleware that logs HTTP requests.
//
// It logs every endpoint hit and includes extra diagnostics for error responses
// (HTTP 4xx/5xx) to simplify root-cause investigation.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		event := logEventByStatus(wrapped.statusCode)
		event.Str("request_id", r.Header.Get("X-Request-ID"))
		event.Str("method", r.Method)
		event.Str("path", r.URL.Path)
		event.Str("query", r.URL.RawQuery)
		event.Int("status", wrapped.statusCode)
		event.Dur("duration", duration)
		event.Str("ip", getClientIP(r))
		event.Str("user_agent", r.UserAgent())

		if wrapped.statusCode >= http.StatusBadRequest {
			addErrorDetails(event, wrapped)
		}

		if wrapped.panicErr != nil {
			event.Interface("panic_error", wrapped.panicErr)
			event.Str("panic_stack", wrapped.panicStack)
		}

		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Int("status", wrapped.statusCode).
			Dur("duration", duration).
			Str("ip", getClientIP(r)).
			Str("user_agent", r.UserAgent()).
			Msg("HTTP Request")
		event.Msg("HTTP request completed")
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
func logEventByStatus(statusCode int) *zerolog.Event {
	switch {
	case statusCode >= http.StatusInternalServerError:
		return log.Error()
	case statusCode >= http.StatusBadRequest:
		return log.Warn()
	default:
		return log.Info()
	}
}

func addErrorDetails(event *zerolog.Event, wrapped *responseWriter) {
	event.Str("status_text", http.StatusText(wrapped.statusCode))
	event.Str("error_reason", errorReason(wrapped.statusCode))
	event.Str("response_body", wrapped.bodyPreview())
}

// responseWriter wraps http.ResponseWriter to capture status code and response body.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       strings.Builder
	panicErr   interface{}
	panicStack string
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(p []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}

	if rw.body.Len() < maxLoggedErrorBody {
		remaining := maxLoggedErrorBody - rw.body.Len()
		if len(p) > remaining {
			_, _ = rw.body.Write(p[:remaining])
		} else {
			_, _ = rw.body.Write(p)
		}
	}

	return rw.ResponseWriter.Write(p)
}

func (rw *responseWriter) bodyPreview() string {
	if rw.body.Len() == 0 {
		return ""
	}

	body := rw.body.String()
	if rw.body.Len() >= maxLoggedErrorBody {
		return body + "...<truncated>"
	}
	return body
}

// SetPanicDetails stores panic metadata captured by Recover middleware.
func (rw *responseWriter) SetPanicDetails(err interface{}, stack string) {
	rw.panicErr = err
	rw.panicStack = stack
}

// Flush implements http.Flusher when the underlying writer supports it.
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack implements http.Hijacker for websocket and raw TCP upgrades.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

// Push implements http.Pusher for HTTP/2 server push.
func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// ReadFrom implements io.ReaderFrom when the underlying writer supports it.
func (rw *responseWriter) ReadFrom(src io.Reader) (int64, error) {
	if rf, ok := rw.ResponseWriter.(io.ReaderFrom); ok {
		return rf.ReadFrom(src)
	}
	return io.Copy(rw.ResponseWriter, src)
}

func errorReason(status int) string {
	switch {
	case status >= http.StatusInternalServerError:
		return "server-side failure: panic, dependency outage, or unhandled internal error"
	case status == http.StatusUnauthorized:
		return "authentication failed: missing/invalid/expired token"
	case status == http.StatusForbidden:
		return "access denied: insufficient permissions or policy restriction"
	case status == http.StatusNotFound:
		return "endpoint/resource not found: wrong route, id, or method"
	case status == http.StatusMethodNotAllowed:
		return "HTTP method is not allowed for this endpoint"
	case status == http.StatusRequestTimeout:
		return "request timed out before handler completed"
	case status == http.StatusTooManyRequests:
		return "rate/usage limit exceeded"
	case status >= http.StatusBadRequest:
		return "client-side validation or request format error"
	default:
		return ""
	}
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP if multiple
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
