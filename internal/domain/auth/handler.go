package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/errorhandler"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// Handler handles auth HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register handles POST /auth/register
// @Summary Регистрация пользователя
// @Description Создает аккаунт для model.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Данные регистрации"
// @Success 201 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorhandler.HandleError(r.Context(), w,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Request body must be valid JSON",
			err)
		return
	}

	// Validate request
	if validationErrors := validator.Validate(&req); validationErrors != nil {
		errorhandler.LogValidationError(r.Context(), validationErrors)
		response.ErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", validationErrors)
		return
	}

	// Register user
	result, err := h.service.Register(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailAlreadyExists):
			errorhandler.HandleError(r.Context(), w,
				http.StatusConflict,
				"EMAIL_ALREADY_EXISTS",
				"Email already registered",
				err)
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"REGISTRATION_FAILED",
				"Failed to register user",
				err)
		}
		return
	}

	response.Created(w, map[string]interface{}{"message": "Registered. Email code sent.", "data": result})
}

// Login handles POST /auth/login
// @Summary Авторизация пользователя
// @Description Выполняет вход пользователя по email/паролю и возвращает access/refresh токены.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Данные для входа"
// @Success 200 {object} response.Response{data=AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} map[string]interface{} "EMAIL_NOT_VERIFIED"
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorhandler.HandleError(r.Context(), w,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Request body must be valid JSON",
			err)
		return
	}

	// Validate request
	if validationErrors := validator.Validate(&req); validationErrors != nil {
		errorhandler.LogValidationError(r.Context(), validationErrors)
		response.ValidationError(w, validationErrors)
		return
	}

	// Login
	result, err := h.service.Login(r.Context(), &req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			errorhandler.HandleError(r.Context(), w,
				http.StatusUnauthorized,
				"INVALID_CREDENTIALS",
				"Invalid email or password",
				err)
		case ErrUserBanned:
			errorhandler.HandleError(r.Context(), w,
				http.StatusForbidden,
				"USER_BANNED",
				"Account is banned",
				err)
		case ErrEmailNotVerified:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    false,
				"error_code": "EMAIL_NOT_VERIFIED",
				"message":    "Email is not verified",
			})
		default:
			errorhandler.HandleError(r.Context(), w,
				http.StatusInternalServerError,
				"LOGIN_FAILED",
				"Failed to login",
				err)
		}
		return
	}

	response.OK(w, result)
}

// Refresh handles POST /auth/refresh
// @Summary Обновление токенов
// @Description Обновляет access/refresh токены по валидному refresh token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} response.Response{data=AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 422 {object} response.Response
// @Router /auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorhandler.HandleError(r.Context(), w,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Request body must be valid JSON",
			err)
		return
	}

	// Validate request
	if validationErrors := validator.Validate(&req); validationErrors != nil {
		errorhandler.LogValidationError(r.Context(), validationErrors)
		response.ValidationError(w, validationErrors)
		return
	}

	// Refresh tokens
	result, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		errorhandler.HandleError(r.Context(), w,
			http.StatusUnauthorized,
			"INVALID_REFRESH_TOKEN",
			"Invalid or expired refresh token",
			err)
		return
	}

	response.OK(w, result)
}

// Logout handles POST /auth/logout
// @Summary Выход пользователя
// @Description Инвалидирует refresh token пользователя.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Security BearerAuth
// @Success 204 {string} string "No Content"
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorhandler.HandleError(r.Context(), w,
			http.StatusBadRequest,
			"INVALID_JSON",
			"Request body must be valid JSON",
			err)
		return
	}

	// Logout (invalidate refresh token)
	_ = h.service.Logout(r.Context(), req.RefreshToken)

	response.NoContent(w)
}

// Me handles GET /auth/me
// @Summary Текущий пользователь
// @Description Возвращает данные авторизованного пользователя.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=UserResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Get current user
	user, err := h.service.GetCurrentUser(r.Context(), userID)
	if err != nil {
		errorhandler.HandleError(r.Context(), w,
			http.StatusNotFound,
			"USER_NOT_FOUND",
			"User not found",
			err)
		return
	}

	response.OK(w, user)
}
