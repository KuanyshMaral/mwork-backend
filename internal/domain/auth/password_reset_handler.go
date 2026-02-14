package auth

import (
	"encoding/json"
	"net/http"

	"github.com/mwork/mwork-api/internal/pkg/response"
)

// PasswordResetHandler handles password reset HTTP requests.
type PasswordResetHandler struct {
	service  *Service
	resetSvc *PasswordResetService
}

func NewPasswordResetHandler(service *Service, resetSvc *PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{service: service, resetSvc: resetSvc}
}

// ForgotPasswordRequest for initiating password reset
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest for setting new password
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// ForgotPassword handles POST /auth/forgot-password
func (h *PasswordResetHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if req.Email == "" {
		response.BadRequest(w, "Email is required")
		return
	}

	user, err := h.service.FindByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		response.OK(w, map[string]string{"message": "If your email is registered, you will receive a reset link"})
		return
	}

	if err := h.resetSvc.SendPasswordResetEmail(r.Context(), user.ID, user.Email, user.Email); err != nil {
		response.OK(w, map[string]string{"message": "If your email is registered, you will receive a reset link"})
		return
	}

	response.OK(w, map[string]string{"message": "If your email is registered, you will receive a reset link"})
}

// ResetPassword handles POST /auth/reset-password
func (h *PasswordResetHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		response.BadRequest(w, "Token and new_password are required")
		return
	}

	if len(req.NewPassword) < 6 {
		response.BadRequest(w, "Password must be at least 6 characters")
		return
	}

	userID, err := h.resetSvc.ValidateResetToken(r.Context(), req.Token)
	if err != nil {
		response.BadRequest(w, "Invalid or expired reset token")
		return
	}

	if err := h.service.UpdatePassword(r.Context(), userID, req.NewPassword); err != nil {
		response.InternalError(w)
		return
	}

	h.resetSvc.InvalidateResetToken(r.Context(), req.Token)
	response.OK(w, map[string]string{"message": "Password reset successfully"})
}
