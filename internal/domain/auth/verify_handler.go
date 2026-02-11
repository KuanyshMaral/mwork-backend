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
	Code string `json:"code"`
}

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
