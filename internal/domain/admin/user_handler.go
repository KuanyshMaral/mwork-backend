package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/mwork/mwork-api/internal/pkg/response"
)

// UserHandler handles admin user management endpoints
type UserHandler struct {
	db       *sqlx.DB
	adminSvc *Service
}

// NewUserHandler creates user handler for admin
func NewUserHandler(db *sqlx.DB, adminSvc *Service) *UserHandler {
	return &UserHandler{db: db, adminSvc: adminSvc}
}

// Routes returns admin user routes
func (h *UserHandler) Routes(jwtSvc *JWTService, adminSvc *Service) chi.Router {
	r := chi.NewRouter()
	r.Use(AuthMiddleware(jwtSvc, adminSvc))

	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Post("/{id}/ban", h.Ban)
	r.Post("/{id}/unban", h.Unban)
	r.Post("/{id}/verify", h.Verify)
	r.Patch("/{id}/status", h.UpdateStatus)

	return r
}

// UserListResponse represents user in admin list
type UserListResponse struct {
	ID                     uuid.UUID `json:"id"`
	Email                  string    `json:"email"`
	Role                   string    `json:"role"`
	EmailVerified          bool      `json:"email_verified"`
	IsBanned               bool      `json:"is_banned"`
	UserVerificationStatus string    `json:"user_verification_status"`
	CreatedAt              string    `json:"created_at"`
	LastLoginAt            *string   `json:"last_login_at,omitempty"`
}

// List handles GET /admin/users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	role := r.URL.Query().Get("role")
	search := r.URL.Query().Get("search")

	// Build query
	query := `SELECT id, email, role, email_verified, is_banned, user_verification_status, created_at, last_login_at 
		FROM users WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM users WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if role != "" && role != "all" {
		query += ` AND role = $` + strconv.Itoa(argIndex)
		countQuery += ` AND role = $` + strconv.Itoa(argIndex)
		args = append(args, role)
		argIndex++
	}

	if search != "" {
		query += ` AND email ILIKE $` + strconv.Itoa(argIndex)
		countQuery += ` AND email ILIKE $` + strconv.Itoa(argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	type userRow struct {
		ID                     uuid.UUID `db:"id"`
		Email                  string    `db:"email"`
		Role                   string    `db:"role"`
		EmailVerified          bool      `db:"email_verified"`
		IsBanned               bool      `db:"is_banned"`
		UserVerificationStatus string    `db:"user_verification_status"`
		CreatedAt              string    `db:"created_at"`
		LastLoginAt            *string   `db:"last_login_at"`
	}

	var users []userRow
	if err := h.db.SelectContext(r.Context(), &users, query, args...); err != nil {
		response.InternalError(w)
		return
	}

	// Get total count
	var total int
	countArgs := args[:len(args)-2] // Remove limit/offset
	if err := h.db.GetContext(r.Context(), &total, countQuery, countArgs...); err != nil {
		total = len(users)
	}

	items := make([]*UserListResponse, len(users))
	for i, u := range users {
		items[i] = &UserListResponse{
			ID:                     u.ID,
			Email:                  u.Email,
			Role:                   u.Role,
			EmailVerified:          u.EmailVerified,
			IsBanned:               u.IsBanned,
			UserVerificationStatus: u.UserVerificationStatus,
			CreatedAt:              u.CreatedAt,
			LastLoginAt:            u.LastLoginAt,
		}
	}

	response.OK(w, map[string]interface{}{
		"users": items,
		"total": total,
	})
}

// GetByID handles GET /admin/users/{id}
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	query := `SELECT id, email, role, email_verified, is_banned, user_verification_status, created_at, last_login_at 
		FROM users WHERE id = $1`

	var user struct {
		ID                     uuid.UUID `db:"id"`
		Email                  string    `db:"email"`
		Role                   string    `db:"role"`
		EmailVerified          bool      `db:"email_verified"`
		IsBanned               bool      `db:"is_banned"`
		UserVerificationStatus string    `db:"user_verification_status"`
		CreatedAt              string    `db:"created_at"`
		LastLoginAt            *string   `db:"last_login_at"`
	}

	if err := h.db.GetContext(r.Context(), &user, query, id); err != nil {
		response.NotFound(w, "User not found")
		return
	}

	response.OK(w, &UserListResponse{
		ID:                     user.ID,
		Email:                  user.Email,
		Role:                   user.Role,
		EmailVerified:          user.EmailVerified,
		IsBanned:               user.IsBanned,
		UserVerificationStatus: user.UserVerificationStatus,
		CreatedAt:              user.CreatedAt,
		LastLoginAt:            user.LastLoginAt,
	})
}

// Ban handles POST /admin/users/{id}/ban
func (h *UserHandler) Ban(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	result, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET is_banned = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		response.InternalError(w)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.NotFound(w, "User not found")
		return
	}

	// Log action
	adminID := GetAdminID(r.Context())
	h.adminSvc.LogActionWithReason(r.Context(), adminID, "user.ban", "user", id, req.Reason, nil, nil)

	response.OK(w, map[string]string{"status": "banned"})
}

// Unban handles POST /admin/users/{id}/unban
func (h *UserHandler) Unban(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	result, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET is_banned = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		response.InternalError(w)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.NotFound(w, "User not found")
		return
	}

	// Log action
	adminID := GetAdminID(r.Context())
	h.adminSvc.logAction(r.Context(), adminID, "user.unban", "user", id, nil, nil)

	response.OK(w, map[string]string{"status": "unbanned"})
}

// Verify handles POST /admin/users/{id}/verify
func (h *UserHandler) Verify(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	result, err := h.db.ExecContext(r.Context(),
		`UPDATE users SET email_verified = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		response.InternalError(w)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.NotFound(w, "User not found")
		return
	}

	// Log action
	adminID := GetAdminID(r.Context())
	h.adminSvc.logAction(r.Context(), adminID, "user.verify", "user", id, nil, nil)

	response.OK(w, map[string]string{"status": "verified"})
}

// UpdateStatus handles PATCH /admin/users/{id}/status
func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	var req struct {
		Status string `json:"status" validate:"required"`
		Reason string `json:"reason,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	// Get admin ID for logging
	adminID := GetAdminID(r.Context())

	// Log before update
	fmt.Printf("UpdateStatus: Updating user %s to status %s by admin %s\n", id, req.Status, adminID)

	// Check current status before update
	var currentStatus struct {
		UserVerificationStatus string  `db:"user_verification_status"`
		VerificationReviewedAt *string `db:"verification_reviewed_at"`
		VerificationReviewedBy *string `db:"verification_reviewed_by"`
	}

	checkQuery := `SELECT user_verification_status, verification_reviewed_at, verification_reviewed_by FROM users WHERE id = $1`
	if err := h.db.GetContext(r.Context(), &currentStatus, checkQuery, id); err == nil {
		fmt.Printf("UpdateStatus: Current status - %s, reviewed_at: %v, reviewed_by: %v\n",
			currentStatus.UserVerificationStatus,
			currentStatus.VerificationReviewedAt,
			currentStatus.VerificationReviewedBy)
	}

	// Build query based on status
	var query string
	var args []interface{}

	if req.Status == "rejected" && req.Reason != "" {
		query = `UPDATE users SET user_verification_status = $1, verification_rejection_reason = $2, verification_reviewed_at = NOW(), verification_reviewed_by = $3, updated_at = NOW() WHERE id = $4`
		args = []interface{}{req.Status, req.Reason, adminID, id}
	} else {
		query = `UPDATE users SET user_verification_status = $1, verification_reviewed_at = NOW(), verification_reviewed_by = $2, updated_at = NOW() WHERE id = $3`
		args = []interface{}{req.Status, adminID, id}
	}

	result, err := h.db.ExecContext(r.Context(), query, args...)
	if err != nil {
		fmt.Printf("UpdateStatus: ERROR updating user - %v\n", err)
		response.InternalError(w)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		fmt.Printf("UpdateStatus: ERROR - no rows affected, user not found\n")
		response.NotFound(w, "User not found")
		return
	}

	// Log after update to verify
	var updatedStatus struct {
		UserVerificationStatus string  `db:"user_verification_status"`
		VerificationReviewedAt *string `db:"verification_reviewed_at"`
		VerificationReviewedBy *string `db:"verification_reviewed_by"`
	}

	if err := h.db.GetContext(r.Context(), &updatedStatus, checkQuery, id); err == nil {
		fmt.Printf("UpdateStatus: After update - status: %s, reviewed_at: %v, reviewed_by: %v\n",
			updatedStatus.UserVerificationStatus,
			updatedStatus.VerificationReviewedAt,
			updatedStatus.VerificationReviewedBy)
	}

	// Log action
	h.adminSvc.LogActionWithReason(r.Context(), adminID, "user.update_status", "user", id, req.Reason, nil, map[string]interface{}{
		"new_status": req.Status,
	})

	response.OK(w, map[string]string{"status": "updated"})
}
