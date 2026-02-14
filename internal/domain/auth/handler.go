package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
	"github.com/rs/zerolog/log"
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
	requestID := middleware.GetRequestID(r.Context())

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errors := validator.Validate(&req); errors != nil {
		response.ErrorWithDetails(w, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", errors)
		return
	}

	// Register user
	result, err := h.service.Register(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailAlreadyExists):
			response.Error(w, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "Email already registered")
		default:
			log.Error().
				Err(err).
				Str("request_id", requestID).
				Str("email", req.Email).
				Msg("failed to register user")
			response.InternalError(w)
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
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Login
	result, err := h.service.Login(r.Context(), &req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			response.Unauthorized(w, "Invalid email or password")
		case ErrUserBanned:
			response.Forbidden(w, "Account is banned")
		case ErrEmailNotVerified:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success":    false,
				"error_code": "EMAIL_NOT_VERIFIED",
				"message":    "Email is not verified",
			})
		default:
			log.Error().
				Err(err).
				Str("email", req.Email).
				Msg("login failed with internal error")
			response.InternalError(w)
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
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Validate request
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	// Refresh tokens
	result, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(w, "Invalid or expired refresh token")
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
		response.BadRequest(w, "Invalid JSON body")
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
		response.NotFound(w, "User not found")
		return
	}

	response.OK(w, user)
}
