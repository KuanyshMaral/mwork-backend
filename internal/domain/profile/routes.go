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
	r.Get("/models/{id}", h.GetModelByID)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		// Create
		r.Post("/models", h.CreateModel)
		r.Post("/employers", h.CreateEmployer)

		// Current User Profile
		r.Get("/me", h.GetMe)

		// Update (Only models implemented fully for now)
		r.Put("/models/{id}", h.UpdateModel)

		// TODO: Add employer list/update routes
	})

	return r
}
