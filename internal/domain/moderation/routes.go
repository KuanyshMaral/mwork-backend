package moderation

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns moderation routes
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// All routes require authentication
	r.Use(authMiddleware)

	// User-facing moderation endpoints
	r.Post("/block", h.BlockUser)
	r.Delete("/block", h.UnblockUser)
	r.Get("/blocks", h.ListBlocks)

	r.Post("/report", h.CreateReport)
	r.Get("/reports/mine", h.ListMyReports)

	return r
}

// AdminRoutes returns admin-only moderation routes
func (h *Handler) AdminRoutes(authMiddleware, adminMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Require auth + admin
	r.Use(authMiddleware)
	r.Use(adminMiddleware)

	r.Get("/reports", h.ListReports)
	r.Post("/reports/{id}/resolve", h.ResolveReport)

	return r
}
