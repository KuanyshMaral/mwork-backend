package lead

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mwork/mwork-api/internal/domain/admin"
)

// Routes returns public lead routes
func (h *Handler) PublicRoutes() chi.Router {
	r := chi.NewRouter()

	// Public endpoint - no auth required
	r.Post("/employer", h.SubmitLead)

	return r
}

// AdminRoutes returns admin lead routes
func (h *Handler) AdminRoutes(jwtSvc *admin.JWTService, adminSvc *admin.Service) chi.Router {
	r := chi.NewRouter()

	// All routes require admin auth
	r.Use(admin.AuthMiddleware(jwtSvc, adminSvc))

	r.Get("/", h.List)
	r.Get("/stats", h.Stats)

	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.GetByID)
		r.Patch("/status", h.UpdateStatus)
		r.Post("/contacted", h.MarkContacted)
		r.Post("/assign", h.Assign)
		r.Post("/convert", h.Convert)
		// Convenience aliases for frontend
		r.Post("/approve", h.Convert) // Same as convert
		r.Post("/reject", h.Reject)   // Reject with reason
	})

	return r
}

// Routes returns legacy routes (backwards compatible)
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	return h.PublicRoutes()
}
