package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/mwork/mwork-api/internal/domain/subscription"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

// UserHandler handles admin user management endpoints
type UserHandler struct {
	db       *sqlx.DB
	adminSvc *Service
	credits  *CreditHandler
	limits   *subscription.Service
}

// NewUserHandler creates user handler for admin
func NewUserHandler(db *sqlx.DB, adminSvc *Service, creditHandler *CreditHandler, limits *subscription.Service) *UserHandler {
	return &UserHandler{db: db, adminSvc: adminSvc, credits: creditHandler, limits: limits}
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
	r.Route("/{id}/credits", func(r chi.Router) {
		r.Use(RequirePermission(PermGrantCredits))
		r.Post("/grant", h.GrantCredits)
		r.Get("/", h.GetUserCredits)
	})
	r.Route("/{id}/limits", func(r chi.Router) {
		r.Use(RequirePermission(PermManageSubscriptions))
		r.Get("/", h.GetUserLimits)
		r.Post("/{limitKey}/adjust", h.AdjustUserLimit)
		r.Post("/{limitKey}/set", h.SetUserLimit)
	})

	return r
}

type adjustLimitRequest struct {
	Delta  int    `json:"delta"`
	Reason string `json:"reason"`
}

type setLimitRequest struct {
	Value  int    `json:"value"`
	Reason string `json:"reason"`
}

func (h *UserHandler) AdjustUserLimit(w http.ResponseWriter, r *http.Request) {
	if h.limits == nil {
		response.InternalError(w)
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}
	limitKey := chi.URLParam(r, "limitKey")
	var req adjustLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}
	actorID := GetAdminID(r.Context())
	status, err := h.limits.AdjustLimit(r.Context(), actorID, userID, limitKey, req.Delta, req.Reason)
	if err != nil {
		switch err {
		case subscription.ErrInvalidLimitKey:
			response.BadRequest(w, "Invalid limit key")
		case subscription.ErrLimitWouldBeNegative:
			response.BadRequest(w, "Final limit cannot be negative")
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, status)
}

func (h *UserHandler) SetUserLimit(w http.ResponseWriter, r *http.Request) {
	if h.limits == nil {
		response.InternalError(w)
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}
	limitKey := chi.URLParam(r, "limitKey")
	var req setLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}
	actorID := GetAdminID(r.Context())
	status, err := h.limits.SetLimit(r.Context(), actorID, userID, limitKey, req.Value, req.Reason)
	if err != nil {
		switch err {
		case subscription.ErrInvalidLimitKey:
			response.BadRequest(w, "Invalid limit key")
		case subscription.ErrLimitWouldBeNegative:
			response.BadRequest(w, "Final limit cannot be negative")
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, status)
}

func (h *UserHandler) GetUserLimits(w http.ResponseWriter, r *http.Request) {
	if h.limits == nil {
		response.InternalError(w)
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}
	items, err := h.limits.GetAllLimitStatuses(r.Context(), userID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// GrantCredits handles POST /admin/users/{id}/credits/grant
func (h *UserHandler) GrantCredits(w http.ResponseWriter, r *http.Request) {
	if h.credits == nil {
		response.InternalError(w)
		return
	}
	h.credits.GrantCredits(w, r)
}
func (h *UserHandler) GetUserCredits(w http.ResponseWriter, r *http.Request) {
	if h.credits == nil {
		response.InternalError(w)
		return
	}
	h.credits.GetUserCredits(w, r)
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
	query := `SELECT id, email, role, email_verified, is_banned, user_verification_status, created_at, last_login_at FROM users WHERE 1=1`
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
	var total int
	countArgs := args[:len(args)-2]
	if err := h.db.GetContext(r.Context(), &total, countQuery, countArgs...); err != nil {
		total = len(users)
	}
	items := make([]*UserListResponse, len(users))
	for i, u := range users {
		items[i] = &UserListResponse{ID: u.ID, Email: u.Email, Role: u.Role, EmailVerified: u.EmailVerified, IsBanned: u.IsBanned, UserVerificationStatus: u.UserVerificationStatus, CreatedAt: u.CreatedAt, LastLoginAt: u.LastLoginAt}
	}
	response.OK(w, map[string]interface{}{"users": items, "total": total})
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}
	query := `SELECT id, email, role, email_verified, is_banned, user_verification_status, created_at, last_login_at FROM users WHERE id = $1`
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
	response.OK(w, &UserListResponse{ID: user.ID, Email: user.Email, Role: user.Role, EmailVerified: user.EmailVerified, IsBanned: user.IsBanned, UserVerificationStatus: user.UserVerificationStatus, CreatedAt: user.CreatedAt, LastLoginAt: user.LastLoginAt})
}

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
	result, err := h.db.ExecContext(r.Context(), `UPDATE users SET is_banned = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		response.InternalError(w)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.NotFound(w, "User not found")
		return
	}
	adminID := GetAdminID(r.Context())
	h.adminSvc.LogActionWithReason(r.Context(), adminID, "user.ban", "user", id, req.Reason, nil, nil)
	response.OK(w, map[string]string{"status": "banned"})
}
func (h *UserHandler) Unban(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}
	result, err := h.db.ExecContext(r.Context(), `UPDATE users SET is_banned = false, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		response.InternalError(w)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.NotFound(w, "User not found")
		return
	}
	adminID := GetAdminID(r.Context())
	h.adminSvc.LogActionWithReason(r.Context(), adminID, "user.unban", "user", id, "", nil, nil)
	response.OK(w, map[string]string{"status": "active"})
}
func (h *UserHandler) Verify(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}
	result, err := h.db.ExecContext(r.Context(), `UPDATE users SET email_verified = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		response.InternalError(w)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.NotFound(w, "User not found")
		return
	}
	adminID := GetAdminID(r.Context())
	h.adminSvc.LogActionWithReason(r.Context(), adminID, "user.verify", "user", id, "", nil, nil)
	response.OK(w, map[string]string{"status": "verified"})
}
func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}
	var req struct {
		IsBanned               *bool   `json:"is_banned,omitempty"`
		EmailVerified          *bool   `json:"email_verified,omitempty"`
		UserVerificationStatus *string `json:"user_verification_status,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}
	setClauses := []string{}
	args := []interface{}{}
	argPos := 1
	if req.IsBanned != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_banned = $%d", argPos))
		args = append(args, *req.IsBanned)
		argPos++
	}
	if req.EmailVerified != nil {
		setClauses = append(setClauses, fmt.Sprintf("email_verified = $%d", argPos))
		args = append(args, *req.EmailVerified)
		argPos++
	}
	if req.UserVerificationStatus != nil {
		setClauses = append(setClauses, fmt.Sprintf("user_verification_status = $%d", argPos))
		args = append(args, *req.UserVerificationStatus)
		argPos++
	}
	if len(setClauses) == 0 {
		response.BadRequest(w, "No fields to update")
		return
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", joinClauses(setClauses, ", "), argPos)
	args = append(args, id)
	result, err := h.db.ExecContext(r.Context(), query, args...)
	if err != nil {
		response.InternalError(w)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		response.NotFound(w, "User not found")
		return
	}
	adminID := GetAdminID(r.Context())
	h.adminSvc.LogActionWithReason(r.Context(), adminID, "user.update_status", "user", id, "", nil, req)
	response.OK(w, map[string]string{"status": "updated"})
}
func joinClauses(clauses []string, sep string) string {
	if len(clauses) == 0 {
		return ""
	}
	out := clauses[0]
	for i := 1; i < len(clauses); i++ {
		out += sep + clauses[i]
	}
	return out
}
