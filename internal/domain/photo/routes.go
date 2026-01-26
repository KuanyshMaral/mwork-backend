package photo

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns photo router
func (h *Handler) Routes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// All routes require authentication
	r.Use(authMiddleware)

	r.Post("/", h.ConfirmUpload)
	r.Delete("/{id}", h.Delete)
	r.Patch("/{id}/avatar", h.SetAvatar)
	r.Patch("/reorder", h.Reorder)

	return r
}

// UploadRoutes returns upload router
func (h *Handler) UploadRoutes(authMiddleware func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Post("/presign", h.Presign)

	return r
}
