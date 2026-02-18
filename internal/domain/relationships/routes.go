package relationships

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns relationships router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// All routes require authentication
	r.Use(authMiddleware)

	// Block/unblock operations
	r.Post("/users/{id}/block", h.BlockUser)
	r.Delete("/users/{id}/block", h.UnblockUser)

	// List blocked users
	r.Get("/users/me/blocked", h.ListBlocked)

	return r
}
