package response

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
)

// DecodeJSON decodes JSON from request body into the provided struct
func DecodeJSON(body io.ReadCloser, v interface{}) error {
	defer body.Close()
	return json.NewDecoder(body).Decode(v)
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo represents error details
type ErrorInfo struct {
	Code       string            `json:"code"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
	ErrorTrace string            `json:"error_trace,omitempty"` // Full error details/stack trace
}

// Meta represents pagination metadata
type Meta struct {
	Total   int  `json:"total"`
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	Pages   int  `json:"pages"`
	HasNext bool `json:"has_next"`
	HasPrev bool `json:"has_prev"`
}

// JSON sends a JSON response
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := Response{
		Success: status >= 200 && status < 300,
		Data:    data,
	}

	json.NewEncoder(w).Encode(resp)
}

// OK sends a 200 OK response
func OK(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, data)
}

// Created sends a 201 Created response
func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, data)
}

// NoContent sends a 204 No Content response
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// WithMeta sends a response with pagination metadata
func WithMeta(w http.ResponseWriter, data interface{}, meta Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := Response{
		Success: true,
		Data:    data,
		Meta:    &meta,
	}

	json.NewEncoder(w).Encode(resp)
}

// Error sends an error response
func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(resp)
}

// ErrorWithDetails sends an error response with details
func ErrorWithDetails(w http.ResponseWriter, status int, code, message string, details map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}

	json.NewEncoder(w).Encode(resp)
}

// BadRequest sends a 400 Bad Request response
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden sends a 403 Forbidden response
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound sends a 404 Not Found response
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, "NOT_FOUND", message)
}

// Conflict sends a 409 Conflict response
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, "CONFLICT", message)
}

// ValidationError sends a 422 Unprocessable Entity response
func ValidationError(w http.ResponseWriter, details map[string]string) {
	ErrorWithDetails(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Validation failed", details)
}

// InternalError sends a 500 Internal Server Error response
func InternalError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
}

// InternalErrorWithError sends a 500 Internal Server Error response with full error details
// Shows the actual error message/stack trace in response
func InternalErrorWithError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	errorTrace := ""
	if err != nil {
		errorTrace = fmt.Sprintf("Error: %v\n\nStack Trace:\n%s", err.Error(), string(debug.Stack()))
	}

	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:       "INTERNAL_ERROR",
			Message:    "Internal server error",
			ErrorTrace: errorTrace,
		},
	}

	json.NewEncoder(w).Encode(resp)
}

// ErrorWithError sends an error response with full error details
// Includes the actual error message and stack trace
func ErrorWithError(w http.ResponseWriter, status int, code string, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorTrace := ""
	if err != nil {
		errorTrace = fmt.Sprintf("Error: %v\n\nStack Trace:\n%s", err.Error(), string(debug.Stack()))
	}

	resp := Response{
		Success: false,
		Error: &ErrorInfo{
			Code:       code,
			Message:    message,
			ErrorTrace: errorTrace,
		},
	}

	json.NewEncoder(w).Encode(resp)
}

// TooManyRequests sends a 429 Too Many Requests response
func TooManyRequests(w http.ResponseWriter) {
	Error(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests, please try again later")
}

// ErrorResponse documents standard API error payload.
// error.code values include: INVALID_PURPOSE, INVALID_CONTENT_TYPE, FILE_TOO_LARGE, UPLOAD_NOT_FOUND,
// UPLOAD_FORBIDDEN, UPLOAD_EXPIRED, UPLOAD_INVALID_STATUS, STORAGE_ERROR.
type ErrorResponse struct {
	Success bool       `json:"success" example:"false"`
	Error   *ErrorInfo `json:"error"`
}
