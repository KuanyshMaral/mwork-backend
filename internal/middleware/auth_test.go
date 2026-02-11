package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/pkg/jwt"
)

func TestAuthMiddlewareAllowsValidAccessToken(t *testing.T) {
	jwtSvc := jwt.NewService("secret", time.Minute, time.Hour)
	token, err := jwtSvc.GenerateAccessToken(uuid.New(), "model", false)
	if err != nil {
		t.Fatalf("token gen failed: %v", err)
	}

	protected := Auth(jwtSvc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
