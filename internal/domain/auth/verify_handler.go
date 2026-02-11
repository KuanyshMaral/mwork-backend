package auth

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

var verifyCodeRegexp = regexp.MustCompile(`^\d{6}$`)

type VerifyConfirmRequest struct {
	Code string `json:"code" pattern:"^\\d{6}$"`
}

// RequestVerify handles POST /auth/verify/request
// @Summary Запрос кода верификации
// @Description Отправляет код верификации пользователю, если аккаунт ещё не подтвержден.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response{data=map[string]string} "status: already_verified|sent"
// @Failure 401 {object} response.Response
// @Router /auth/verify/request [post]
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

	if status == "already_verified" {
		response.OK(w, map[string]string{"status": "already_verified"})
		return
	}

	response.OK(w, map[string]string{"status": "sent"})
}

// ConfirmVerify handles POST /auth/verify/confirm
// @Summary Подтверждение верификации
// @Description Проверяет 6-значный код верификации и подтверждает аккаунт.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body VerifyConfirmRequest true "Код подтверждения"
// @Success 200 {object} response.Response{data=map[string]string} "status: verified"
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /auth/verify/confirm [post]
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
		response.BadRequest(w, "Invalid verification code")
		return
	}

	status, err := h.service.ConfirmVerificationCode(r.Context(), userID, req.Code)
	if err != nil {
		if err == ErrInvalidVerificationCode {
			response.BadRequest(w, "Invalid code")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{"status": status})
}
