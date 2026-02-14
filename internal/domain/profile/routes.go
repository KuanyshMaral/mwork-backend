package profile

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns profile router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes for models
	r.Get("/models", h.ListModels)
	r.Get("/models/promoted", h.ListPromotedModels)
	r.Get("/models/{id}", h.GetModelByID)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		// Current User Profile
		r.Get("/me", h.GetMe)

		// Update
		r.Put("/models/{id}", h.UpdateModel)
		r.Put("/employers/{id}", h.UpdateEmployer)
	})

	return r
}
