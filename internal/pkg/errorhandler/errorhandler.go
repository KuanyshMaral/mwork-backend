package errorhandler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/pkg/logger"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// HandleError handles an error response with full logging
// It logs the error details and sends a formatted error response with full error trace
func HandleError(ctx context.Context, w http.ResponseWriter, status int, code, message string, err error) {
	// Log the error with full context
	event := log.Error().
		Str("request_id", getRequestID(ctx)).
		Str("error_code", code).
		Str("error_message", message).
		Int("status_code", status)

	if err != nil {
		event.Err(err)
	}

	event.Msg("Request error")

	// Send error response with full error details and stack trace
	response.ErrorWithError(w, status, code, message, err)
}

// HandleErrorWithDetails handles an error response with additional details and logging
func HandleErrorWithDetails(ctx context.Context, w http.ResponseWriter, status int, code, message string, details map[string]string, err error) {
	// Log the error with details
	event := log.Error().
		Str("request_id", getRequestID(ctx)).
		Str("error_code", code).
		Str("error_message", message).
		Int("status_code", status)

	if err != nil {
		event.Err(err)
	}

	if details != nil {
		event.Interface("error_details", details)
	}

	event.Msg("Request error with details")

	// Send error response with details
	response.ErrorWithDetails(w, status, code, message, details)
}

// HandlePanicError logs and handles panics with full stack trace
func HandlePanicError(ctx context.Context, w http.ResponseWriter, panicErr interface{}, stackTrace string) {
	log.Error().
		Str("request_id", getRequestID(ctx)).
		Interface("panic_error", panicErr).
		Str("panic_stack", stackTrace).
		Msg("Request panic error")

	// Send error response with full panic details (stack trace)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	resp := response.Response{
		Success: false,
		Error: &response.ErrorInfo{
			Code:       "PANIC_ERROR",
			Message:    "Internal server panic",
			ErrorTrace: stackTrace,
		},
	}

	json.NewEncoder(w).Encode(resp)
}

// LogRequest logs HTTP request details including body for debugging
func LogRequest(ctx context.Context, r *http.Request, body string) {
	logger.LogInfo(ctx, "HTTP request",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"request_id", getRequestID(ctx),
	)

	// Log body only for certain content types and methods that use body
	if body != "" && (r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch) {
		logger.LogDebug(ctx, "Request body",
			"body", truncateString(body, 1000),
		)
	}
}

// LogResponse logs HTTP response details
func LogResponse(ctx context.Context, w http.ResponseWriter, status int, body string) {
	logger.LogInfo(ctx, "HTTP response",
		"status", status,
		"request_id", getRequestID(ctx),
	)

	// Log response body for errors (status >= 400)
	if status >= 400 && body != "" {
		logger.LogDebug(ctx, "Error response body",
			"body", truncateString(body, 2048),
		)
	}
}

// LogDatabaseError logs database errors with context
func LogDatabaseError(ctx context.Context, operation string, err error, query string) {
	log.Error().
		Str("request_id", getRequestID(ctx)).
		Str("operation", operation).
		Str("query", query).
		Err(err).
		Msg("Database error")
}

// LogValidationError logs validation errors with details
func LogValidationError(ctx context.Context, fieldErrors map[string]string) {
	errJSON, _ := json.Marshal(fieldErrors)
	log.Warn().
		Str("request_id", getRequestID(ctx)).
		RawJSON("validation_errors", errJSON).
		Msg("Validation error")
}

// LogExternalServiceError logs errors from external service calls
func LogExternalServiceError(ctx context.Context, service string, endpoint string, statusCode int, err error, body string) {
	log.Error().
		Str("request_id", getRequestID(ctx)).
		Str("external_service", service).
		Str("endpoint", endpoint).
		Int("status_code", statusCode).
		Err(err).
		Str("response_body", truncateString(body, 1000)).
		Msg("External service error")
}

// Helper functions

func getRequestID(ctx context.Context) string {
	if reqID := ctx.Value("request_id"); reqID != nil {
		if id, ok := reqID.(string); ok {
			return id
		}
	}
	return "unknown"
}

func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "...<truncated>"
	}
	return s
}
