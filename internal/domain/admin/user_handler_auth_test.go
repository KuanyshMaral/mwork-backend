package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserLimitsEndpoint_UnauthorizedWithoutToken(t *testing.T) {
	h := NewUserHandler(nil, nil, &CreditHandler{}, nil)
	r := h.Routes(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/123e4567-e89b-12d3-a456-426614174000/limits/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestRequirePermission_ForbiddenWithoutRole(t *testing.T) {
	mw := RequirePermission(PermManageSubscriptions)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
