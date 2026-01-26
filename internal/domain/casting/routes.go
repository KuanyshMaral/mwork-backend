package casting

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns casting router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", h.Create)
		r.Get("/my", h.ListMy)
		r.Put("/{id}", h.Update)
		r.Patch("/{id}/status", h.UpdateStatus)
		r.Delete("/{id}", h.Delete)
	})

	return r
}
