package auth

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// VerificationHandler handles email verification and password reset HTTP requests
type VerificationHandler struct {
	service   *Service
	verifySvc *VerificationService
}

// NewVerificationHandler creates verification handler
func NewVerificationHandler(service *Service, verifySvc *VerificationService) *VerificationHandler {
	return &VerificationHandler{
		service:   service,
		verifySvc: verifySvc,
	}
}

// SendVerificationRequest for sending verification code
type SendVerificationRequest struct {
	Email string `json:"email"`
}

// VerifyEmailRequest for verifying email
type VerifyEmailRequest struct {
	Code string `json:"code" validate:"required,len=6"`
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

// SendVerification handles POST /auth/send-verification
// @Summary Отправка кода верификации email
// @Description Отправляет код подтверждения на email текущего пользователя.
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/send-verification [post]
// Sends a verification code to the authenticated user's email
func (h *VerificationHandler) SendVerification(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	// Get user info
	user, err := h.service.GetCurrentUser(r.Context(), userID)
	if err != nil {
		response.NotFound(w, "User not found")
		return
	}

	// Check if already verified
	if user.EmailVerified {
		response.BadRequest(w, "Email already verified")
		return
	}

	// Send verification email
	if err := h.verifySvc.SendVerificationEmail(r.Context(), userID, user.Email, user.Email); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{
		"message": "Verification code sent to your email",
	})
}

// VerifyEmail handles POST /auth/verify-email
// @Summary Подтверждение email
// @Description Проверяет код подтверждения и отмечает email как подтвержденный.
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body VerifyEmailRequest true "Код подтверждения"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/verify-email [post]
// Verifies the email with the code
func (h *VerificationHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == uuid.Nil {
		response.Unauthorized(w, "Authentication required")
		return
	}

	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if req.Code == "" || len(req.Code) != 6 {
		response.BadRequest(w, "Invalid verification code")
		return
	}

	// Verify the code
	valid, err := h.verifySvc.VerifyEmail(r.Context(), userID, req.Code)
	if err != nil {
		response.InternalError(w)
		return
	}

	if !valid {
		response.BadRequest(w, "Invalid or expired verification code")
		return
	}

	// Mark email as verified in database
	if err := h.service.MarkEmailVerified(r.Context(), userID); err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, map[string]string{
		"message": "Email verified successfully",
	})
}

// ForgotPassword handles POST /auth/forgot-password
// @Summary Запрос на сброс пароля
// @Description Отправляет инструкцию по сбросу пароля на email.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Email пользователя"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /auth/forgot-password [post]
// Sends a password reset email
func (h *VerificationHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if req.Email == "" {
		response.BadRequest(w, "Email is required")
		return
	}

	// Find user by email
	user, err := h.service.FindByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		// Don't reveal if email exists or not
		response.OK(w, map[string]string{
			"message": "If your email is registered, you will receive a reset link",
		})
		return
	}

	// Send reset email
	if err := h.verifySvc.SendPasswordResetEmail(r.Context(), user.ID, user.Email, user.Email); err != nil {
		// Log error but don't reveal to user
		response.OK(w, map[string]string{
			"message": "If your email is registered, you will receive a reset link",
		})
		return
	}

	response.OK(w, map[string]string{
		"message": "If your email is registered, you will receive a reset link",
	})
}

// ResetPassword handles POST /auth/reset-password
// @Summary Сброс пароля
// @Description Устанавливает новый пароль по reset token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body ResetPasswordRequest true "Токен и новый пароль"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /auth/reset-password [post]
// Resets password using the token
func (h *VerificationHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
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

	// Validate token and get user ID
	userID, err := h.verifySvc.ValidateResetToken(r.Context(), req.Token)
	if err != nil {
		response.BadRequest(w, "Invalid or expired reset token")
		return
	}

	// Update password
	if err := h.service.UpdatePassword(r.Context(), userID, req.NewPassword); err != nil {
		response.InternalError(w)
		return
	}

	// Invalidate token after use
	h.verifySvc.InvalidateResetToken(r.Context(), req.Token)

	response.OK(w, map[string]string{
		"message": "Password reset successfully",
	})
}
