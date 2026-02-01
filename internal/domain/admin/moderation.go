package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/mwork/mwork-api/internal/pkg/response"
	"github.com/mwork/mwork-api/internal/pkg/validator"
)

// ModerationHandler handles content moderation
type ModerationHandler struct {
	db       *sqlx.DB
	adminSvc *Service
}

// NewModerationHandler creates moderation handler
func NewModerationHandler(db *sqlx.DB, adminSvc *Service) *ModerationHandler {
	return &ModerationHandler{
		db:       db,
		adminSvc: adminSvc,
	}
}

// --- User Management ---

// ListUsers handles GET /admin/users
func (h *ModerationHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := 50
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

	query := `
		SELECT u.id, u.email, u.role, u.is_banned, u.is_verified, u.created_at,
		       p.first_name, p.last_name, p.avatar_url
		FROM users u
		LEFT JOIN profiles p ON u.id = p.user_id
		ORDER BY u.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := h.db.QueryContext(r.Context(), query, limit, offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var email, role string
		var isBanned, isVerified sql.NullBool
		var createdAt time.Time
		var firstName, lastName, avatarURL sql.NullString

		if err := rows.Scan(&id, &email, &role, &isBanned, &isVerified, &createdAt, &firstName, &lastName, &avatarURL); err != nil {
			continue
		}

		users = append(users, map[string]interface{}{
			"id":          id,
			"email":       email,
			"role":        role,
			"is_banned":   isBanned.Bool,
			"is_verified": isVerified.Bool,
			"created_at":  createdAt.Format(time.RFC3339),
			"first_name":  firstName.String,
			"last_name":   lastName.String,
			"avatar_url":  avatarURL.String,
		})
	}

	var total int
	h.db.GetContext(r.Context(), &total, `SELECT COUNT(*) FROM users`)

	response.OK(w, map[string]interface{}{
		"items": users,
		"total": total,
	})
}

// BanUser handles PATCH /admin/users/{id}/ban
func (h *ModerationHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	var req BanUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	adminID := GetAdminID(r.Context())

	var query string
	if req.IsBanned {
		query = `UPDATE users SET is_banned = true, banned_at = NOW(), banned_reason = $2, banned_by = $3 WHERE id = $1`
	} else {
		query = `UPDATE users SET is_banned = false, banned_at = NULL, banned_reason = NULL, banned_by = NULL WHERE id = $1`
	}

	_, err = h.db.ExecContext(r.Context(), query, userID, req.Reason, adminID)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Audit log
	action := "user.unban"
	if req.IsBanned {
		action = "user.ban"
	}
	h.adminSvc.LogActionWithReason(r.Context(), adminID, action, "user", userID, req.Reason, nil, map[string]bool{"is_banned": req.IsBanned})

	response.OK(w, map[string]string{"status": "ok"})
}

// VerifyUser handles PATCH /admin/users/{id}/verify
func (h *ModerationHandler) VerifyUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	var req VerifyUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	adminID := GetAdminID(r.Context())

	if req.IsVerified != nil {
		_, err = h.db.ExecContext(r.Context(),
			`UPDATE users SET is_verified = $2, verified_at = CASE WHEN $2 THEN NOW() ELSE NULL END, verified_by = CASE WHEN $2 THEN $3 ELSE NULL END WHERE id = $1`,
			userID, *req.IsVerified, adminID)
	}
	if req.IsIdentityVerified != nil {
		_, err = h.db.ExecContext(r.Context(),
			`UPDATE users SET is_identity_verified = $2, verified_at = CASE WHEN $2 THEN NOW() ELSE verified_at END, verified_by = CASE WHEN $2 THEN $3 ELSE verified_by END WHERE id = $1`,
			userID, *req.IsIdentityVerified, adminID)
	}

	if err != nil {
		response.InternalError(w)
		return
	}

	h.adminSvc.LogActionWithReason(r.Context(), adminID, "user.verify", "user", userID, "", nil, req)

	response.OK(w, map[string]string{"status": "ok"})
}

// --- Profile Moderation ---

// ListPendingProfiles handles GET /admin/profiles
func (h *ModerationHandler) ListPendingProfiles(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}

	query := `
		SELECT p.*, u.email 
		FROM profiles p
		JOIN users u ON p.user_id = u.id
		WHERE p.moderation_status = $1
		ORDER BY p.created_at DESC
		LIMIT 50
	`

	rows, err := h.db.QueryContext(r.Context(), query, status)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer rows.Close()

	var profiles []map[string]interface{}
	for rows.Next() {
		// Simplified - just return raw data
		profiles = append(profiles, map[string]interface{}{"status": status})
	}

	response.OK(w, profiles)
}

// ModerateProfile handles PATCH /admin/profiles/{id}/moderate
func (h *ModerationHandler) ModerateProfile(w http.ResponseWriter, r *http.Request) {
	profileID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid profile ID")
		return
	}

	var req ModerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	if errors := validator.Validate(&req); errors != nil {
		response.ValidationError(w, errors)
		return
	}

	adminID := GetAdminID(r.Context())

	_, err = h.db.ExecContext(r.Context(),
		`UPDATE profiles SET moderation_status = $2, moderated_at = NOW(), moderated_by = $3, moderation_note = $4 WHERE id = $1`,
		profileID, req.Status, adminID, req.Note)
	if err != nil {
		response.InternalError(w)
		return
	}

	h.adminSvc.LogActionWithReason(r.Context(), adminID, "profile.moderate", "profile", profileID, req.Note,
		nil, map[string]string{"status": req.Status})

	response.OK(w, map[string]string{"status": "ok"})
}

// --- Photo Moderation ---

// ListPendingPhotos handles GET /admin/photos/pending
func (h *ModerationHandler) ListPendingPhotos(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT ph.id, ph.profile_id, ph.url, ph.moderation_status, ph.created_at,
		       p.first_name, p.last_name
		FROM photos ph
		JOIN profiles p ON ph.profile_id = p.id
		WHERE ph.moderation_status = 'pending'
		ORDER BY ph.created_at ASC
		LIMIT 50
	`

	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer rows.Close()

	var photos []map[string]interface{}
	for rows.Next() {
		var id, profileID uuid.UUID
		var url, status string
		var createdAt time.Time
		var firstName, lastName sql.NullString

		if err := rows.Scan(&id, &profileID, &url, &status, &createdAt, &firstName, &lastName); err != nil {
			continue
		}

		photos = append(photos, map[string]interface{}{
			"id":         id,
			"profile_id": profileID,
			"url":        url,
			"status":     status,
			"created_at": createdAt.Format(time.RFC3339),
			"owner_name": firstName.String + " " + lastName.String,
		})
	}

	response.OK(w, photos)
}

// ModeratePhoto handles PATCH /admin/photos/{id}/moderate
func (h *ModerationHandler) ModeratePhoto(w http.ResponseWriter, r *http.Request) {
	photoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid photo ID")
		return
	}

	var req struct {
		Status string `json:"status" validate:"required,oneof=approved rejected"`
		IsNSFW bool   `json:"is_nsfw"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	adminID := GetAdminID(r.Context())

	_, err = h.db.ExecContext(r.Context(),
		`UPDATE photos SET moderation_status = $2, moderated_at = NOW(), moderated_by = $3, is_nsfw = $4 WHERE id = $1`,
		photoID, req.Status, adminID, req.IsNSFW)
	if err != nil {
		response.InternalError(w)
		return
	}

	h.adminSvc.LogActionWithReason(r.Context(), adminID, "photo.moderate", "photo", photoID, "",
		nil, req)

	response.OK(w, map[string]string{"status": "ok"})
}

// DeletePhoto handles DELETE /admin/photos/{id}
func (h *ModerationHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	photoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid photo ID")
		return
	}

	adminID := GetAdminID(r.Context())

	_, err = h.db.ExecContext(r.Context(), `DELETE FROM photos WHERE id = $1`, photoID)
	if err != nil {
		response.InternalError(w)
		return
	}

	h.adminSvc.LogActionWithReason(r.Context(), adminID, "photo.delete", "photo", photoID, "", nil, nil)

	response.OK(w, map[string]string{"status": "deleted"})
}

// --- Casting Moderation ---

// ListCastings handles GET /admin/castings
func (h *ModerationHandler) ListCastings(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	query := `
		SELECT c.id, c.title, c.status, c.moderation_status, c.is_featured, c.created_at,
		       u.email as employer_email
		FROM castings c
		JOIN users u ON c.employer_id = u.id
		ORDER BY c.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := h.db.QueryContext(r.Context(), query, limit, offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer rows.Close()

	var castings []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var title, status string
		var modStatus sql.NullString
		var isFeatured sql.NullBool
		var createdAt time.Time
		var employerEmail string

		if err := rows.Scan(&id, &title, &status, &modStatus, &isFeatured, &createdAt, &employerEmail); err != nil {
			continue
		}

		castings = append(castings, map[string]interface{}{
			"id":                id,
			"title":             title,
			"status":            status,
			"moderation_status": modStatus.String,
			"is_featured":       isFeatured.Bool,
			"created_at":        createdAt.Format(time.RFC3339),
			"employer_email":    employerEmail,
		})
	}

	response.OK(w, castings)
}

// FeatureCasting handles PATCH /admin/castings/{id}/feature
func (h *ModerationHandler) FeatureCasting(w http.ResponseWriter, r *http.Request) {
	castingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid casting ID")
		return
	}

	var req struct {
		IsFeatured bool `json:"is_featured"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	adminID := GetAdminID(r.Context())

	_, err = h.db.ExecContext(r.Context(),
		`UPDATE castings SET is_featured = $2 WHERE id = $1`,
		castingID, req.IsFeatured)
	if err != nil {
		response.InternalError(w)
		return
	}

	h.adminSvc.LogActionWithReason(r.Context(), adminID, "casting.feature", "casting", castingID, "",
		nil, req)

	response.OK(w, map[string]string{"status": "ok"})
}

// --- Subscription Management ---

// ListSubscriptions handles GET /admin/subscriptions
func (h *ModerationHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT s.id, s.user_id, s.plan_id, s.status, s.started_at, s.expires_at,
		       u.email
		FROM subscriptions s
		JOIN users u ON s.user_id = u.id
		ORDER BY s.created_at DESC
		LIMIT 50
	`

	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer rows.Close()

	var subs []map[string]interface{}
	for rows.Next() {
		var id, userID uuid.UUID
		var planID, status, email string
		var startedAt time.Time
		var expiresAt sql.NullTime

		if err := rows.Scan(&id, &userID, &planID, &status, &startedAt, &expiresAt, &email); err != nil {
			continue
		}

		sub := map[string]interface{}{
			"id":         id,
			"user_id":    userID,
			"plan_id":    planID,
			"status":     status,
			"started_at": startedAt.Format(time.RFC3339),
			"email":      email,
		}
		if expiresAt.Valid {
			sub["expires_at"] = expiresAt.Time.Format(time.RFC3339)
		}
		subs = append(subs, sub)
	}

	response.OK(w, subs)
}

// UpgradeUser handles POST /admin/subscriptions/{userId}/upgrade
func (h *ModerationHandler) UpgradeUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	var req struct {
		PlanID string `json:"plan_id" validate:"required,oneof=pro agency"`
		Days   int    `json:"days" validate:"required,min=1,max=365"`
		Reason string `json:"reason" validate:"required"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}

	adminID := GetAdminID(r.Context())
	now := time.Now()
	expiresAt := now.AddDate(0, 0, req.Days)

	// Cancel existing active subscription
	h.db.ExecContext(r.Context(), `UPDATE subscriptions SET status = 'cancelled' WHERE user_id = $1 AND status = 'active'`, userID)

	// Create new subscription
	subID := uuid.New()
	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO subscriptions (id, user_id, plan_id, started_at, expires_at, status, billing_period, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 'active', 'monthly', $4, $4)`,
		subID, userID, req.PlanID, now, expiresAt)
	if err != nil {
		response.InternalError(w)
		return
	}

	h.adminSvc.LogActionWithReason(r.Context(), adminID, "subscription.upgrade", "user", userID, req.Reason,
		nil, map[string]interface{}{"plan_id": req.PlanID, "days": req.Days})

	response.Created(w, map[string]interface{}{
		"subscription_id": subID,
		"plan_id":         req.PlanID,
		"expires_at":      expiresAt.Format(time.RFC3339),
	})
}

// --- Organization Verification ---

type organizationRow struct {
	ID                 uuid.UUID      `db:"id"`
	LegalName          string         `db:"legal_name"`
	BrandName          sql.NullString `db:"brand_name"`
	BinIIN             string         `db:"bin_iin"`
	OrgType            string         `db:"org_type"`
	City               sql.NullString `db:"city"`
	Phone              sql.NullString `db:"phone"`
	Email              sql.NullString `db:"email"`
	Website            sql.NullString `db:"website"`
	ContactPerson      sql.NullString `db:"contact_person"`
	ContactPhone       sql.NullString `db:"contact_phone"`
	ContactTelegram    sql.NullString `db:"contact_telegram"`
	ContactWhatsApp    sql.NullString `db:"contact_whatsapp"`
	VerificationStatus string         `db:"verification_status"`
	VerificationNotes  sql.NullString `db:"verification_notes"`
	RejectionReason    sql.NullString `db:"rejection_reason"`
	VerifiedAt         sql.NullTime   `db:"verified_at"`
	VerifiedBy         uuid.NullUUID  `db:"verified_by"`
	CreatedAt          time.Time      `db:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at"`
}

func strPtr(ns sql.NullString) *string {
	if ns.Valid {
		s := ns.String
		return &s
	}
	return nil
}

func timePtr(nt sql.NullTime) *string {
	if nt.Valid {
		s := nt.Time.Format(time.RFC3339)
		return &s
	}
	return nil
}

func uuidPtr(nu uuid.NullUUID) *uuid.UUID {
	if nu.Valid {
		u := nu.UUID
		return &u
	}
	return nil
}

func orgResp(row organizationRow) OrganizationResponse {
	return OrganizationResponse{
		ID:                 row.ID,
		LegalName:          row.LegalName,
		BrandName:          strPtr(row.BrandName),
		BinIIN:             row.BinIIN,
		OrgType:            row.OrgType,
		City:               strPtr(row.City),
		Phone:              strPtr(row.Phone),
		Email:              strPtr(row.Email),
		Website:            strPtr(row.Website),
		ContactPerson:      strPtr(row.ContactPerson),
		ContactPhone:       strPtr(row.ContactPhone),
		ContactTelegram:    strPtr(row.ContactTelegram),
		ContactWhatsApp:    strPtr(row.ContactWhatsApp),
		VerificationStatus: row.VerificationStatus,
		VerificationNotes:  strPtr(row.VerificationNotes),
		RejectionReason:    strPtr(row.RejectionReason),
		VerifiedAt:         timePtr(row.VerifiedAt),
		VerifiedBy:         uuidPtr(row.VerifiedBy),
		CreatedAt:          row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          row.UpdatedAt.Format(time.RFC3339),
	}
}

// ListOrganizations handles GET /admin/moderation/organizations
func (h *ModerationHandler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	limit := 50
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

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	switch status {
	case "none", "pending", "in_review", "verified", "rejected":
	default:
		response.BadRequest(w, "Invalid status")
		return
	}

	rows := []organizationRow{}
	q := `SELECT id, legal_name, brand_name, bin_iin, org_type, city, phone, email, website,
		contact_person, contact_phone, contact_telegram, contact_whatsapp,
		verification_status, verification_notes, rejection_reason, verified_at, verified_by,
		created_at, updated_at
		FROM organizations
		WHERE verification_status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	if err := h.db.SelectContext(r.Context(), &rows, q, status, limit, offset); err != nil {
		response.InternalError(w)
		return
	}

	var total int
	_ = h.db.GetContext(r.Context(), &total, `SELECT COUNT(*) FROM organizations WHERE verification_status = $1`, status)

	out := make([]OrganizationResponse, len(rows))
	for i, row := range rows {
		out[i] = orgResp(row)
	}

	response.OK(w, ListOrganizationsResponse{Organizations: out, Total: total})
}

// GetOrganization handles GET /admin/moderation/organizations/{id}
func (h *ModerationHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid organization ID")
		return
	}

	var row organizationRow
	q := `SELECT id, legal_name, brand_name, bin_iin, org_type, city, phone, email, website,
		contact_person, contact_phone, contact_telegram, contact_whatsapp,
		verification_status, verification_notes, rejection_reason, verified_at, verified_by,
		created_at, updated_at
		FROM organizations WHERE id = $1`
	if err := h.db.GetContext(r.Context(), &row, q, orgID); err != nil {
		if err == sql.ErrNoRows {
			response.NotFound(w, "Organization not found")
			return
		}
		response.InternalError(w)
		return
	}

	response.OK(w, orgResp(row))
}

// VerifyOrganization handles PATCH /admin/moderation/organizations/{id}/verify
func (h *ModerationHandler) VerifyOrganization(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "Invalid organization ID")
		return
	}

	var req VerifyOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid JSON body")
		return
	}
	if errs := validator.Validate(&req); errs != nil {
		response.ValidationError(w, errs)
		return
	}

	adminID := GetAdminID(r.Context())

	notes := sql.NullString{String: req.Notes, Valid: req.Notes != ""}
	rej := sql.NullString{String: req.RejectionReason, Valid: req.RejectionReason != ""}

	tx, err := h.db.BeginTxx(r.Context(), nil)
	if err != nil {
		response.InternalError(w)
		return
	}
	defer tx.Rollback()

	// Update organization verification
	_, err = tx.ExecContext(r.Context(),
		`UPDATE organizations
		SET verification_status = $2,
			verification_notes = $3,
			rejection_reason = $4,
			verified_at = CASE WHEN $2 = 'verified' THEN NOW() ELSE NULL END,
			verified_by = CASE WHEN $2 IN ('verified','rejected','in_review') THEN $5 ELSE NULL END,
			updated_at = NOW()
		WHERE id = $1`,
		orgID, req.Status, notes, rej, adminID)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Sync employer/agency user verification status for this organization
	_, err = tx.ExecContext(r.Context(),
		`UPDATE users
		SET user_verification_status = $2,
			verification_reviewed_at = NOW(),
			verification_reviewed_by = $3,
			verification_notes = $4,
			verification_rejection_reason = $5
		WHERE organization_id = $1 AND role IN ('employer','agency')`,
		orgID, req.Status, adminID, notes, rej)
	if err != nil {
		response.InternalError(w)
		return
	}

	// Sync employer profile flag
	_, err = tx.ExecContext(r.Context(),
		`UPDATE employer_profiles
		SET is_verified = ($2 = 'verified'),
			verified_at = CASE WHEN $2 = 'verified' THEN NOW() ELSE NULL END
		WHERE user_id IN (SELECT id FROM users WHERE organization_id = $1 AND role IN ('employer','agency'))`,
		orgID, req.Status)
	if err != nil {
		response.InternalError(w)
		return
	}

	if err := tx.Commit(); err != nil {
		response.InternalError(w)
		return
	}

	action := "organization.status"
	switch req.Status {
	case "verified":
		action = "organization.approve"
	case "rejected":
		action = "organization.reject"
	case "in_review":
		action = "organization.in_review"
	case "pending":
		action = "organization.pending"
	}

	reason := req.RejectionReason
	if reason == "" {
		reason = req.Notes
	}
	h.adminSvc.LogActionWithReason(r.Context(), adminID, action, "organization", orgID, reason, nil, map[string]interface{}{"status": req.Status})

	response.OK(w, map[string]string{"status": "ok"})
}

// ModerationRoutes returns moderation router
func (h *ModerationHandler) Routes(jwtSvc *JWTService, adminSvc *Service) chi.Router {
	r := chi.NewRouter()
	r.Use(AuthMiddleware(jwtSvc, adminSvc))

	// Users
	r.Route("/users", func(r chi.Router) {
		r.Use(RequirePermission(PermViewUsers))
		r.Get("/", h.ListUsers)

		r.Group(func(r chi.Router) {
			r.Use(RequirePermission(PermBanUsers))
			r.Patch("/{id}/ban", h.BanUser)
		})
		r.Group(func(r chi.Router) {
			r.Use(RequirePermission(PermVerifyUsers))
			r.Patch("/{id}/verify", h.VerifyUser)
		})
	})

	// Organizations
	r.Route("/organizations", func(r chi.Router) {
		r.Use(RequirePermission(PermViewOrganizations))
		r.Get("/", h.ListOrganizations)
		r.Get("/{id}", h.GetOrganization)

		r.Group(func(r chi.Router) {
			r.Use(RequirePermission(PermVerifyOrganizations))
			r.Patch("/{id}/verify", h.VerifyOrganization)
		})
	})

	// Profiles
	r.Route("/profiles", func(r chi.Router) {
		r.Use(RequirePermission(PermModerateContent))
		r.Get("/", h.ListPendingProfiles)
		r.Patch("/{id}/moderate", h.ModerateProfile)
	})

	// Photos
	r.Route("/photos", func(r chi.Router) {
		r.Use(RequirePermission(PermModerateContent))
		r.Get("/pending", h.ListPendingPhotos)
		r.Patch("/{id}/moderate", h.ModeratePhoto)
		r.Delete("/{id}", h.DeletePhoto)
	})

	// Castings
	r.Route("/castings", func(r chi.Router) {
		r.Use(RequirePermission(PermViewContent))
		r.Get("/", h.ListCastings)
		r.Patch("/{id}/feature", h.FeatureCasting)
	})

	// Subscriptions
	r.Route("/subscriptions", func(r chi.Router) {
		r.Use(RequirePermission(PermViewSubscriptions))
		r.Get("/", h.ListSubscriptions)

		r.Group(func(r chi.Router) {
			r.Use(RequirePermission(PermManageSubscriptions))
			r.Post("/{userId}/upgrade", h.UpgradeUser)
		})
	})

	return r
}

// Helper for impersonation (generate user token)
func (h *ModerationHandler) ImpersonateUser(ctx context.Context, adminID, userID uuid.UUID) {
	// This would generate a user JWT for debugging purposes
	// Implementation depends on user JWT service
	h.adminSvc.LogActionWithReason(ctx, adminID, "user.impersonate", "user", userID, "Debug session", nil, nil)
}
