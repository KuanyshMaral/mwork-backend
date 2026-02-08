package photostudio_booking

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns PhotoStudio booking router.
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes (no auth required)
	r.Get("/studios", h.GetStudios)

	// Protected routes (auth required)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/bookings", h.CreateBooking)
	})

	return r
}
