package subscription

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/middleware"
)

func TestGetModelCastingLimits_EmployerForbidden(t *testing.T) {
	h := NewHandler(&Service{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, middleware.UserIDKey, uuid.New())
	ctx = context.WithValue(ctx, middleware.RoleKey, "employer")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.GetModelCastingLimits(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}
