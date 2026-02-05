package admin

import (
	"github.com/go-chi/chi/v5"
)

// Routes returns admin router
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Auth routes (no auth required)
	r.Post("/auth/login", h.Login)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware(h.jwtSvc, h.service))

		// Current admin
		r.Get("/auth/me", h.Me)

		// Admin management (super_admin only)
		r.Route("/admins", func(r chi.Router) {
			r.Use(RequirePermission(PermManageAdmins))
			r.Get("/", h.ListAdmins)
			r.Post("/", h.CreateAdmin)
			r.Patch("/{id}", h.UpdateAdmin)
		})

		// Feature flags
		r.Route("/features", func(r chi.Router) {
			r.Use(RequirePermission(PermManageFeatures))
			r.Get("/", h.ListFeatures)
			r.Patch("/{key}", h.UpdateFeature)
		})

		// Analytics
		r.Route("/analytics", func(r chi.Router) {
			r.Use(RequirePermission(PermViewAnalytics))
			r.Get("/dashboard", h.Dashboard)
			r.Get("/revenue", h.Revenue)
		})

		// Audit logs
		r.Route("/audit", func(r chi.Router) {
			r.Use(RequirePermission(PermViewAuditLogs))
			r.Get("/logs", h.AuditLogs)
		})

		// Reports moderation
		r.Route("/reports", func(r chi.Router) {
			r.Use(RequirePermission(PermModerateContent))
			r.Get("/", h.ListReports)
			r.Patch("/{id}/status", h.ResolveReport)
		})

		// User management
		r.Route("/users", func(r chi.Router) {
			r.Get("/", h.ListUsers)
			r.Patch("/{id}/status", h.UpdateUserStatus)

			// B3: Credit management endpoints (requires credits.grant permission)
			r.Route("/{id}/credits", func(r chi.Router) {
				r.Use(RequirePermission(PermGrantCredits))
				r.Post("/grant", h.creditHandler.GrantCredits)
				r.Get("/", h.creditHandler.GetUserCredits)
			})
		})

		// PhotoStudio sync
		r.Route("/photostudio", func(r chi.Router) {
			r.Use(RequirePermission(PermViewUsers))
			r.Post("/resync", h.ResyncPhotoStudioUsers)
		})

		// SQL execution (super admin only - for temporary operations)
		r.Post("/sql", h.ExecuteSql)

	})

	return r
}
