package response

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns response router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// All routes require authentication
	r.Use(authMiddleware)

	r.Get("/my", h.ListMyApplications)
	r.Patch("/{id}/status", h.UpdateStatus)

	return r
}

// CastingResponseRoutes returns routes for casting responses
func (h *Handler) CastingResponseRoutes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Post("/", h.Apply)
	r.Get("/", h.ListByCasting)

	return r
}
