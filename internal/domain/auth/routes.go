package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns auth router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes (no auth required)
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)

	// Protected routes (auth required)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/logout", h.Logout)
		r.Get("/me", h.Me)
		r.Post("/verify/request", h.RequestVerify)
		r.Post("/verify/confirm", h.ConfirmVerify)
	})

	return r
}
