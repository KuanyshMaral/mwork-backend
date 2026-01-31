package organization

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns organization router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes
	r.Get("/{id}", h.Get)
	r.Get("/{id}/castings", h.GetCastings)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)

		// Members management
		r.Get("/{id}/members", h.GetMembers)
		r.Post("/{id}/members/invite", h.InviteMember)
		r.Patch("/{id}/members/{memberId}", h.UpdateMemberRole)
		r.Delete("/{id}/members/{memberId}", h.RemoveMember)

		// Follow system
		r.Post("/{id}/follow", h.Follow)
		r.Delete("/{id}/follow", h.Unfollow)
		r.Get("/{id}/follow/check", h.CheckFollowing)
	})

	return r
}
