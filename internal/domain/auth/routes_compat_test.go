package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/domain/user"
)

func TestPublicVerifyRequestWithoutAuthorizationIsNot401(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "public-verify@example.com", EmailVerified: false}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	codeRepo := &fakeVerificationCodeRepo{rec: &VerificationCodeRecord{UserID: u.ID, Attempts: 0, ExpiresAt: time.Now().Add(time.Minute)}}
	svc := NewService(repo, &fakeModelProfileRepo{}, nil, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, codeRepo, "pepper", false, false, nil)
	h := NewHandler(svc)

	authRequired := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"UNAUTHORIZED"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	router := h.Routes(authRequired)

	body, _ := json.Marshal(map[string]string{"email": u.Email})
	req := httptest.NewRequest(http.MethodPost, "/verify/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code == http.StatusUnauthorized {
		t.Fatalf("expected public verify endpoint to bypass auth middleware, got 401: %s", rr.Body.String())
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestDeprecatedVerifyMeWithoutAuthorizationReturns401(t *testing.T) {
	repo := &fakeUserRepo{}
	svc := NewService(repo, &fakeModelProfileRepo{}, nil, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, &fakeVerificationCodeRepo{}, "pepper", false, false, nil)
	h := NewHandler(svc)

	authRequired := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":"UNAUTHORIZED"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	router := h.Routes(authRequired)
	req := httptest.NewRequest(http.MethodPost, "/verify/request/me", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for deprecated protected verify endpoint without auth, got %d body=%s", rr.Code, rr.Body.String())
	}
}
