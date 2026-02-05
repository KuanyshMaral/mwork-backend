package admin

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/pkg/photostudio"
	"github.com/mwork/mwork-api/internal/pkg/response"
)

const (
	defaultResyncLimit = 500
	maxResyncLimit     = 1000
	resyncRPS          = 5
)

// PhotoStudioClient defines PhotoStudio sync client interface.
type PhotoStudioClient interface {
	SyncUser(ctx context.Context, payload photostudio.SyncUserPayload) error
}

// PhotoStudioHandler handles admin PhotoStudio operations.
type PhotoStudioHandler struct {
	db      *sqlx.DB
	client  PhotoStudioClient
	enabled bool
	timeout time.Duration
}

// NewPhotoStudioHandler creates a PhotoStudio admin handler.
func NewPhotoStudioHandler(db *sqlx.DB, client PhotoStudioClient, enabled bool, timeout time.Duration) *PhotoStudioHandler {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &PhotoStudioHandler{
		db:      db,
		client:  client,
		enabled: enabled,
		timeout: timeout,
	}
}

// ResyncUsers handles POST /api/v1/admin/photostudio/resync
func (h *PhotoStudioHandler) ResyncUsers(w http.ResponseWriter, r *http.Request) {
	if !h.enabled || h.client == nil {
		response.BadRequest(w, "photostudio sync is disabled")
		return
	}

	limit := defaultResyncLimit
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			if v > maxResyncLimit {
				v = maxResyncLimit
			}
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	const query = `SELECT id, email, role FROM users ORDER BY created_at ASC LIMIT $1 OFFSET $2`
	var users []struct {
		ID    string `db:"id"`
		Email string `db:"email"`
		Role  string `db:"role"`
	}
	if err := h.db.SelectContext(r.Context(), &users, query, limit, offset); err != nil {
		response.InternalError(w)
		return
	}

	processed := 0
	success := 0
	failed := 0

	interval := time.Second / resyncRPS
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for _, u := range users {
		<-ticker.C
		processed++

		payload := photostudio.SyncUserPayload{
			MWorkUserID: u.ID,
			Email:       u.Email,
			Role:        u.Role,
		}

		syncCtx, cancel := context.WithTimeout(r.Context(), h.timeout)
		err := h.client.SyncUser(syncCtx, payload)
		cancel()

		if err != nil {
			failed++
			log.Warn().
				Err(err).
				Str("user_id", payload.MWorkUserID).
				Str("email", payload.Email).
				Str("role", payload.Role).
				Int("processed", processed).
				Int("success", success).
				Int("failed", failed).
				Msg("photostudio resync failed")
			continue
		}

		success++
		log.Info().
			Str("user_id", payload.MWorkUserID).
			Str("email", payload.Email).
			Str("role", payload.Role).
			Int("processed", processed).
			Int("success", success).
			Int("failed", failed).
			Msg("photostudio resync ok")
	}

	response.OK(w, map[string]int{
		"processed": processed,
		"success":   success,
		"failed":    failed,
		"limit":     limit,
		"offset":    offset,
	})
}
