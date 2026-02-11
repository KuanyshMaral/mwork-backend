package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/pkg/jwt"
	"github.com/mwork/mwork-api/internal/pkg/password"
	"github.com/mwork/mwork-api/internal/pkg/photostudio"
)

type fakeUserRepo struct {
	created *user.User
	byEmail *user.User
	byID    *user.User
}

func (f *fakeUserRepo) Create(ctx context.Context, u *user.User) error { f.created = u; return nil }
func (f *fakeUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if f.byID != nil {
		return f.byID, nil
	}
	if f.byEmail != nil && f.byEmail.ID == id {
		return f.byEmail, nil
	}
	return nil, nil
}
func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if f.byEmail != nil && f.byEmail.Email == email {
		return f.byEmail, nil
	}
	return nil, nil
}
func (f *fakeUserRepo) Update(ctx context.Context, u *user.User) error { return nil }
func (f *fakeUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeUserRepo) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	if f.byEmail != nil && f.byEmail.ID == id {
		f.byEmail.EmailVerified = verified
		f.byEmail.IsVerified = verified
	}
	if f.byID != nil && f.byID.ID == id {
		f.byID.EmailVerified = verified
		f.byID.IsVerified = verified
	}
	return nil
}
func (f *fakeUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return nil
}
func (f *fakeUserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status user.Status) error {
	return nil
}
func (f *fakeUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	return nil
}

type fakePhotoStudioClient struct {
	called chan photostudio.SyncUserPayload
	err    error
}

type fakeModelProfileRepo struct{}

func (f *fakeModelProfileRepo) Create(ctx context.Context, profile *ModelProfile) error { return nil }
func (f *fakeModelProfileRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*ModelProfile, error) {
	return nil, nil
}

type fakeEmployerProfileRepo struct{}

func (f *fakeEmployerProfileRepo) Create(ctx context.Context, profile *EmployerProfile) error {
	return nil
}
func (f *fakeEmployerProfileRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*EmployerProfile, error) {
	return nil, nil
}

func (f *fakePhotoStudioClient) SyncUser(ctx context.Context, payload photostudio.SyncUserPayload) error {
	if f.called != nil {
		f.called <- payload
	}
	return f.err
}

type fakeRefreshRepo struct {
	mu    sync.Mutex
	items map[string]*RefreshTokenRecord
}

func newFakeRefreshRepo() *fakeRefreshRepo {
	return &fakeRefreshRepo{items: map[string]*RefreshTokenRecord{}}
}

func (r *fakeRefreshRepo) Create(ctx context.Context, rec *RefreshTokenRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyRec := *rec
	r.items[rec.TokenHash] = &copyRec
	return nil
}

func (r *fakeRefreshRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshTokenRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.items[tokenHash]
	if !ok {
		return nil, errors.New("not found")
	}
	copyRec := *rec
	return &copyRec, nil
}

func (r *fakeRefreshRepo) MarkUsed(ctx context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.items[tokenHash]; ok {
		rec.UsedAt.Time = time.Now()
		rec.UsedAt.Valid = true
	}
	return nil
}
func (r *fakeRefreshRepo) RevokeByTokenHash(ctx context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.items[tokenHash]; ok {
		rec.RevokedAt.Time = time.Now()
		rec.RevokedAt.Valid = true
	}
	return nil
}
func (r *fakeRefreshRepo) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.items {
		if rec.UserID == userID {
			rec.RevokedAt.Time = time.Now()
			rec.RevokedAt.Valid = true
		}
	}
	return nil
}

func TestRegisterIgnoresPhotoStudioError(t *testing.T) {
	repo := &fakeUserRepo{}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	called := make(chan photostudio.SyncUserPayload, 1)

	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, &fakePhotoStudioClient{called: called, err: errors.New("boom")}, true, 50*time.Millisecond, nil, "test-pepper", false)

	resp, err := svc.Register(context.Background(), &RegisterRequest{Email: "user@example.com", Password: "password123", Role: "model"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected auth response")
	}

	select {
	case <-called:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected photostudio sync to be attempted")
	}
}

func TestLoginAllowsUnverifiedUser(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "user3@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now()}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false)

	resp, err := svc.Login(context.Background(), &LoginRequest{Email: "user3@example.com", Password: "password123"})
	if err != nil || resp == nil {
		t.Fatalf("expected login success, err=%v", err)
	}
	if resp.User.IsVerified {
		t.Fatal("expected unverified user in response")
	}
}

func TestRefreshRotationInvalidatesOldToken(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "rot@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now()}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	refreshRepo := newFakeRefreshRepo()
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, refreshRepo, &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false)

	loginResp, err := svc.Login(context.Background(), &LoginRequest{Email: u.Email, Password: "password123"})
	if err != nil {
		t.Fatalf("login err: %v", err)
	}
	oldRefresh := loginResp.Tokens.RefreshToken

	refreshResp, err := svc.Refresh(context.Background(), oldRefresh)
	if err != nil {
		t.Fatalf("refresh err: %v", err)
	}
	if refreshResp.Tokens.RefreshToken == oldRefresh {
		t.Fatal("expected refresh token rotation")
	}

	if _, err := svc.Refresh(context.Background(), oldRefresh); err != ErrInvalidRefreshToken {
		t.Fatalf("expected old refresh to be invalid, got %v", err)
	}
}

func TestRefreshExpiredReturnsUnauthorizedError(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "exp@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now()}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, -time.Minute)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false)

	loginResp, err := svc.Login(context.Background(), &LoginRequest{Email: u.Email, Password: "password123"})
	if err != nil {
		t.Fatalf("login err: %v", err)
	}

	if _, err := svc.Refresh(context.Background(), loginResp.Tokens.RefreshToken); err != ErrInvalidRefreshToken {
		t.Fatalf("expected invalid refresh for expired token, got %v", err)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "logout@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now()}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	refreshRepo := newFakeRefreshRepo()
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, refreshRepo, &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false)

	loginResp, err := svc.Login(context.Background(), &LoginRequest{Email: u.Email, Password: "password123"})
	if err != nil {
		t.Fatalf("login err: %v", err)
	}
	if err := svc.Logout(context.Background(), loginResp.Tokens.RefreshToken); err != nil {
		t.Fatalf("logout err: %v", err)
	}
	if _, err := svc.Refresh(context.Background(), loginResp.Tokens.RefreshToken); err != ErrInvalidRefreshToken {
		t.Fatalf("expected revoked refresh to be invalid, got %v", err)
	}
}
