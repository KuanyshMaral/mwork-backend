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
	r.Post("/verify/request", h.RequestVerifyPublic)
	r.Post("/verify/confirm", h.ConfirmVerifyPublic)

	// Protected routes (auth required)
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/logout", h.Logout)
		r.Get("/me", h.Me)
		// Deprecated protected endpoints kept for backward compatibility
		r.Post("/verify/request/me", h.RequestVerify)
		r.Post("/verify/confirm/me", h.ConfirmVerify)
	})

	return r
}
