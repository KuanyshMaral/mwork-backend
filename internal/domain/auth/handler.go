package auth

import (
	"encoding/json"
	"io"
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
// @Description Создает аккаунт для model/employer или agency в зависимости от поля role.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Данные регистрации"
// @Success 201 {object} response.Response{data=AuthResponse}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 422 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	// First, parse role to determine request type
	var roleCheck struct {
		Role string `json:"role"`
	}

	// Read body into buffer to parse twice
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Parse role first
	if err := json.Unmarshal(bodyBytes, &roleCheck); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Role-based parsing and registration
	switch roleCheck.Role {
	case "agency":
		var req AgencyRegisterRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			response.BadRequest(w, "Invalid JSON body")
			return
		}

		// Validate agency fields
		if errors := validator.Validate(&req); errors != nil {
			response.ValidationError(w, errors)
			return
		}

		// Register agency user
		result, err := h.service.RegisterAgency(r.Context(), &req)
		if err != nil {
			switch err {
			case ErrEmailAlreadyExists:
				response.Conflict(w, "Email already registered")
			default:
				log.Error().
					Err(err).
					Str("email", req.Email).
					Str("role", "agency").
					Msg("failed to register agency user")
				response.InternalError(w)
			}
			return
		}

		response.Created(w, result)

	case "model", "employer":
		var req RegisterRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			response.BadRequest(w, "Invalid JSON body")
			return
		}

		// Validate request
		if errors := validator.Validate(&req); errors != nil {
			response.ValidationError(w, errors)
			return
		}

		// Register user
		result, err := h.service.Register(r.Context(), &req)
		if err != nil {
			switch err {
			case ErrEmailAlreadyExists:
				response.Conflict(w, "Email already registered")
			case ErrInvalidRole:
				response.BadRequest(w, "Role must be 'model' or 'employer'")
			default:
				log.Error().
					Err(err).
					Str("email", req.Email).
					Str("role", req.Role).
					Msg("failed to register user")
				response.InternalError(w)
			}
			return
		}

		response.Created(w, result)

	default:
		response.BadRequest(w, "invalid role")
	}
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
// @Failure 403 {object} response.Response
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
// @Success 204 {string} string "No Content"
// @Failure 400 {object} response.Response
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
// @Router /auth/me [get]
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
