package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

type fakeEmailGuardUserRepo struct {
	byID *user.User
}

func (f *fakeEmailGuardUserRepo) Create(context.Context, *user.User) error { return nil }
func (f *fakeEmailGuardUserRepo) GetByID(context.Context, uuid.UUID) (*user.User, error) {
	return f.byID, nil
}
func (f *fakeEmailGuardUserRepo) GetByEmail(context.Context, string) (*user.User, error) {
	return nil, nil
}
func (f *fakeEmailGuardUserRepo) Update(context.Context, *user.User) error { return nil }
func (f *fakeEmailGuardUserRepo) Delete(context.Context, uuid.UUID) error  { return nil }
func (f *fakeEmailGuardUserRepo) UpdateEmailVerified(context.Context, uuid.UUID, bool) error {
	return nil
}
func (f *fakeEmailGuardUserRepo) UpdateVerificationFlags(context.Context, uuid.UUID, bool, bool) error {
	return nil
}
func (f *fakeEmailGuardUserRepo) UpdatePassword(context.Context, uuid.UUID, string) error { return nil }
func (f *fakeEmailGuardUserRepo) UpdateStatus(context.Context, uuid.UUID, user.Status) error {
	return nil
}
func (f *fakeEmailGuardUserRepo) UpdateLastLogin(context.Context, uuid.UUID, string) error {
	return nil
}

func TestRequireVerifiedEmailBlocksUnverifiedProtectedEndpoint(t *testing.T) {
	uid := uuid.New()
	repo := &fakeEmailGuardUserRepo{byID: &user.User{ID: uid, EmailVerified: false}}
	guard := RequireVerifiedEmail(repo, []string{"/api/v1/auth/login"})

	h := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/me", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, uid))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRequireVerifiedEmailAllowsWhitelistedVerifyRoute(t *testing.T) {
	uid := uuid.New()
	repo := &fakeEmailGuardUserRepo{byID: &user.User{ID: uid, EmailVerified: false}}
	guard := RequireVerifiedEmail(repo, []string{"/api/v1/auth/verify/request"})

	h := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify/request", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, uid))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequireVerifiedEmailAllowsVerifiedProtectedEndpoint(t *testing.T) {
	uid := uuid.New()
	repo := &fakeEmailGuardUserRepo{byID: &user.User{ID: uid, EmailVerified: true}}
	guard := RequireVerifiedEmail(repo, nil)

	h := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/castings", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, uid))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
