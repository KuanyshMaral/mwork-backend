package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

var verifyCodeRegexp = regexp.MustCompile(`^\d{6}$`)

type VerifyConfirmRequest struct {
	Code string `json:"code" pattern:"^\\d{6}$"`
}

// RequestVerifyPublic handles POST /auth/verify/request
// @Summary Запрос кода верификации (public)
// @Description Отправляет код верификации по email без раскрытия существования пользователя.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body VerifyRequestPublicRequest true "Email"
// @Success 200 {object} response.Response{data=map[string]string} "status: already_verified|sent"
// @Failure 400 {object} response.Response
// @Router /auth/verify/request [post]
func (h *Handler) RequestVerifyPublic(w http.ResponseWriter, r *http.Request) {
	var req VerifyRequestPublicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	u, err := h.service.FindByEmail(r.Context(), normalizeEmail(req.Email))
	if err != nil {
		response.InternalError(w)
		return
	}
	if u == nil {
		response.OK(w, map[string]string{"status": "sent"})
		return
	}

	status, err := h.service.RequestVerificationCode(r.Context(), u.ID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.OK(w, map[string]string{"status": "sent"})
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": status})
}

// ConfirmVerifyPublic handles POST /auth/verify/confirm
// @Summary Подтверждение верификации (public)
// @Description Подтверждает email по паре email+code.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body VerifyConfirmPublicRequest true "Email и код подтверждения"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.Response
// @Failure 429 {object} response.Response
// @Router /auth/verify/confirm [post]
func (h *Handler) ConfirmVerifyPublic(w http.ResponseWriter, r *http.Request) {
	var req VerifyConfirmPublicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}
	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}
	if !verifyCodeRegexp.MatchString(req.Code) {
		response.Error(w, http.StatusBadRequest, "INVALID_CODE", "Invalid code")
		return
	}

	u, err := h.service.FindByEmail(r.Context(), normalizeEmail(req.Email))
	if err != nil {
		response.InternalError(w)
		return
	}
	if u == nil {
		response.Error(w, http.StatusBadRequest, "INVALID_CODE", "Invalid code")
		return
	}

	status, err := h.service.ConfirmVerificationCode(r.Context(), u.ID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidVerificationCode):
			response.Error(w, http.StatusBadRequest, "INVALID_CODE", "Invalid code")
		case errors.Is(err, ErrTooManyAttempts):
			response.Error(w, http.StatusTooManyRequests, "TOO_MANY_ATTEMPTS", "Too many attempts")
		default:
			response.InternalError(w)
		}
		return
	}

	userResp, err := h.service.GetCurrentUser(r.Context(), u.ID)
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]interface{}{"status": status, "user": userResp})
}

// RequestVerify handles deprecated protected POST /auth/verify/request/me
// @Summary Запрос кода верификации (deprecated)
// @Description Deprecated: используйте публичный endpoint /auth/verify/request с email.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Deprecated
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 401 {object} response.Response
func (h *Handler) RequestVerify(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	status, err := h.service.RequestVerificationCode(r.Context(), userID)
	if err != nil {
		if err == ErrUserNotFound {
			response.NotFound(w, "User not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": status})
}

// ConfirmVerify handles deprecated protected POST /auth/verify/confirm/me
// @Summary Подтверждение верификации (deprecated)
// @Description Deprecated: используйте публичный endpoint /auth/verify/confirm с email+code.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Deprecated
// @Param request body VerifyConfirmRequest true "Код подтверждения"
// @Success 200 {object} response.Response{data=map[string]string}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 429 {object} response.Response
func (h *Handler) ConfirmVerify(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	var req VerifyConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}
	if !verifyCodeRegexp.MatchString(req.Code) {
		response.Error(w, http.StatusBadRequest, "INVALID_CODE", "Invalid code")
		return
	}

	status, err := h.service.ConfirmVerificationCode(r.Context(), userID, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidVerificationCode):
			response.Error(w, http.StatusBadRequest, "INVALID_CODE", "Invalid code")
		case errors.Is(err, ErrTooManyAttempts):
			response.Error(w, http.StatusTooManyRequests, "TOO_MANY_ATTEMPTS", "Too many attempts")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, map[string]string{"status": status})
}
