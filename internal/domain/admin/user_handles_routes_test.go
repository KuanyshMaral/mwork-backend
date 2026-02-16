package admin

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestUserHandlerRoutes_RegistersCreditsEndpoints(t *testing.T) {
	h := NewUserHandler(nil, nil, &CreditHandler{})
	r := h.Routes(nil, nil)

	patterns := map[string]bool{}
	if err := chi.Walk(r, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		patterns[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	if !patterns["POST /{id}/credits/grant"] {
		t.Fatalf("expected POST /{id}/credits/grant to be registered")
	}
	if !patterns["GET /{id}/credits/"] {
		t.Fatalf("expected GET /{id}/credits/ to be registered")
	}
}
