package experience

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes registers work experience routes
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes
	r.Get("/{id}/experience", h.List)

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/{id}/experience", h.Create)
		r.Delete("/{id}/experience/{expId}", h.Delete)
	})

	return r
}
