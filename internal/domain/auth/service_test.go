package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"

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

func (f *fakeUserRepo) Create(ctx context.Context, u *user.User) error {
	f.created = u
	f.byID = u
	f.byEmail = u
	return nil
}
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
	}
	if f.byID != nil && f.byID.ID == id {
		f.byID.EmailVerified = verified
	}
	return nil
}
func (f *fakeUserRepo) UpdateVerificationFlags(ctx context.Context, id uuid.UUID, emailVerified bool, isVerified bool) error {
	if f.byEmail != nil && f.byEmail.ID == id {
		f.byEmail.EmailVerified = emailVerified
		f.byEmail.IsVerified = isVerified
	}
	if f.byID != nil && f.byID.ID == id {
		f.byID.EmailVerified = emailVerified
		f.byID.IsVerified = isVerified
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

type conflictOnCreateUserRepo struct{ fakeUserRepo }

func (f *conflictOnCreateUserRepo) Create(ctx context.Context, u *user.User) error {
	return &pq.Error{Code: pq.ErrorCode("23505"), Constraint: "users_email_key", Table: "users", Column: "email", Message: "duplicate key value violates unique constraint \"users_email_key\""}
}

func TestRegisterSuccess(t *testing.T) {
	repo := &fakeUserRepo{}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, &fakeVerificationCodeRepo{}, "pepper", false, false, nil)

	resp, err := svc.Register(context.Background(), &RegisterRequest{Email: "new@example.com", Password: "password123"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected auth response")
	}
	if repo.created == nil {
		t.Fatal("expected user to be created")
	}
	if repo.created.EmailVerified {
		t.Fatal("expected new user email_verified=false")
	}
	if resp.User.EmailVerified {
		t.Fatal("expected response email_verified=false")
	}
	if resp.User.IsVerified {
		t.Fatal("expected response is_verified alias=false for unverified email")
	}
}

func TestRegisterMapsDuplicateEmailToDomainError(t *testing.T) {
	repo := &conflictOnCreateUserRepo{}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)

	_, err := svc.Register(context.Background(), &RegisterRequest{Email: "duplicate@example.com", Password: "password123"})
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestRegisterIgnoresPhotoStudioError(t *testing.T) {
	repo := &fakeUserRepo{}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	called := make(chan photostudio.SyncUserPayload, 1)

	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, &fakePhotoStudioClient{called: called, err: errors.New("boom")}, true, 50*time.Millisecond, &fakeVerificationCodeRepo{}, "test-pepper", false, false, nil)

	resp, err := svc.Register(context.Background(), &RegisterRequest{Email: "user@example.com", Password: "password123"})
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

func TestLoginUnverifiedReturnsError(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "user3@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now(), EmailVerified: false, IsVerified: false}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)

	resp, err := svc.Login(context.Background(), &LoginRequest{Email: "user3@example.com", Password: "password123"})
	if err != ErrEmailNotVerified {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}
	if resp != nil {
		t.Fatal("expected nil response for unverified login")
	}
}

func TestLoginVerifiedReturnsTokens(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "verified@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now(), EmailVerified: true, IsVerified: true}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)

	resp, err := svc.Login(context.Background(), &LoginRequest{Email: u.Email, Password: "password123"})
	if err != nil {
		t.Fatalf("expected login success, err=%v", err)
	}
	if resp == nil || resp.Tokens.AccessToken == "" || resp.Tokens.RefreshToken == "" {
		t.Fatal("expected tokens for verified login")
	}
}

func TestRefreshRotationInvalidatesOldToken(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "rot@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now(), EmailVerified: true, IsVerified: true}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	refreshRepo := newFakeRefreshRepo()
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, refreshRepo, &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)

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

func TestRefreshWithOpaqueTokenRotationRevokesOld(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "opaque@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now(), EmailVerified: true, IsVerified: true}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	refreshRepo := newFakeRefreshRepo()
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, refreshRepo, &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)

	loginResp, err := svc.Login(context.Background(), &LoginRequest{Email: u.Email, Password: "password123"})
	if err != nil {
		t.Fatalf("login err: %v", err)
	}
	oldRefresh := loginResp.Tokens.RefreshToken
	if len(oldRefresh) != 64 {
		t.Fatalf("expected opaque token length 64, got %d", len(oldRefresh))
	}
	oldHash := svc.hashRefreshToken(oldRefresh)

	refreshResp, err := svc.Refresh(context.Background(), oldRefresh)
	if err != nil {
		t.Fatalf("refresh err: %v", err)
	}
	if len(refreshResp.Tokens.RefreshToken) != 64 || refreshResp.Tokens.RefreshToken == oldRefresh {
		t.Fatal("expected rotated opaque refresh token")
	}
	old := refreshRepo.items[oldHash]
	if old == nil || !old.RevokedAt.Valid {
		t.Fatal("expected old opaque token to be revoked")
	}
}

func TestRefreshWithLegacyJWTDeniedWhenLegacyFallbackDisabled(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "legacy-off@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now(), EmailVerified: true, IsVerified: true}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	refreshRepo := newFakeRefreshRepo()
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, refreshRepo, &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)

	legacyToken, jti, expiresAt, err := jwtService.GenerateRefreshToken(u.ID)
	if err != nil {
		t.Fatalf("generate legacy token err: %v", err)
	}
	legacyHash := jwt.HashRefreshToken(legacyToken)
	if err := refreshRepo.Create(context.Background(), &RefreshTokenRecord{ID: uuid.New(), UserID: u.ID, TokenHash: legacyHash, JTI: jti, ExpiresAt: expiresAt}); err != nil {
		t.Fatalf("create legacy refresh record err: %v", err)
	}

	if _, err := svc.Refresh(context.Background(), legacyToken); err != ErrInvalidRefreshToken {
		t.Fatalf("expected ErrInvalidRefreshToken when legacy fallback disabled, got %v", err)
	}
}

func TestRefreshWithLegacyJWTReturnsOpaqueToken(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "legacy@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now(), EmailVerified: true, IsVerified: true}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	refreshRepo := newFakeRefreshRepo()
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, refreshRepo, &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, true, nil)

	legacyToken, jti, expiresAt, err := jwtService.GenerateRefreshToken(u.ID)
	if err != nil {
		t.Fatalf("generate legacy token err: %v", err)
	}
	legacyHash := jwt.HashRefreshToken(legacyToken)
	if err := refreshRepo.Create(context.Background(), &RefreshTokenRecord{ID: uuid.New(), UserID: u.ID, TokenHash: legacyHash, JTI: jti, ExpiresAt: expiresAt}); err != nil {
		t.Fatalf("create legacy refresh record err: %v", err)
	}

	resp, err := svc.Refresh(context.Background(), legacyToken)
	if err != nil {
		t.Fatalf("refresh err: %v", err)
	}
	if len(resp.Tokens.RefreshToken) != 64 || strings.Count(resp.Tokens.RefreshToken, ".") != 0 {
		t.Fatalf("expected opaque refresh token, got %q", resp.Tokens.RefreshToken)
	}
	old := refreshRepo.items[legacyHash]
	if old == nil || !old.RevokedAt.Valid {
		t.Fatal("expected legacy token to be revoked")
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "logout@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now(), EmailVerified: true, IsVerified: true}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	refreshRepo := newFakeRefreshRepo()
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, refreshRepo, &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)

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

type fakeVerificationCodeRepo struct {
	rec             *VerificationCodeRecord
	incrementResult int
	incrementCount  int
	markUsedCount   int
	invalidateCount int
}

func (f *fakeVerificationCodeRepo) Upsert(ctx context.Context, userID uuid.UUID, codeHash string, expiresAt time.Time) error {
	return nil
}
func (f *fakeVerificationCodeRepo) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*VerificationCodeRecord, error) {
	if f.rec == nil {
		return nil, errors.New("not found")
	}
	cp := *f.rec
	return &cp, nil
}
func (f *fakeVerificationCodeRepo) IncrementAttempts(ctx context.Context, userID uuid.UUID) (int, error) {
	f.incrementCount++
	if f.rec != nil {
		f.rec.Attempts++
	}
	if f.incrementResult > 0 {
		return f.incrementResult, nil
	}
	if f.rec != nil {
		return f.rec.Attempts, nil
	}
	return 1, nil
}
func (f *fakeVerificationCodeRepo) Invalidate(ctx context.Context, userID uuid.UUID) error {
	f.invalidateCount++
	return nil
}
func (f *fakeVerificationCodeRepo) MarkUsed(ctx context.Context, userID uuid.UUID) error {
	f.markUsedCount++
	return nil
}

func TestConfirmVerificationCodeWrongCodeIncrementsAttempts(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "u@example.com"}
	repo := &fakeUserRepo{byID: u}
	codeRepo := &fakeVerificationCodeRepo{rec: &VerificationCodeRecord{UserID: u.ID, CodeHash: "expected", Attempts: 0, ExpiresAt: time.Now().Add(time.Minute)}}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, codeRepo, "pepper", false, false, nil)

	_, err := svc.ConfirmVerificationCode(context.Background(), u.ID, "111111")
	if !errors.Is(err, ErrInvalidVerificationCode) {
		t.Fatalf("expected ErrInvalidVerificationCode, got %v", err)
	}
	if codeRepo.incrementCount != 1 {
		t.Fatalf("expected increment attempts once, got %d", codeRepo.incrementCount)
	}
}

func TestConfirmVerificationCodeReturnsTooManyAttemptsAfterFive(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "u@example.com"}
	repo := &fakeUserRepo{byID: u}
	codeRepo := &fakeVerificationCodeRepo{rec: &VerificationCodeRecord{UserID: u.ID, CodeHash: "expected", Attempts: 4, ExpiresAt: time.Now().Add(time.Minute)}, incrementResult: 5}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, codeRepo, "pepper", false, false, nil)

	_, err := svc.ConfirmVerificationCode(context.Background(), u.ID, "111111")
	if !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("expected ErrTooManyAttempts, got %v", err)
	}
	if codeRepo.invalidateCount != 1 {
		t.Fatalf("expected invalidate called once, got %d", codeRepo.invalidateCount)
	}
}

func TestConfirmVerificationCodeSuccessMarksUsedAndFlags(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "u@example.com", IsVerified: false, EmailVerified: false}
	repo := &fakeUserRepo{byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)

	hash := svc.hashVerificationCode("123456")
	codeRepo := &fakeVerificationCodeRepo{rec: &VerificationCodeRecord{UserID: u.ID, CodeHash: hash, Attempts: 0, ExpiresAt: time.Now().Add(time.Minute)}}
	svc.verificationCodeRepo = codeRepo

	status, err := svc.ConfirmVerificationCode(context.Background(), u.ID, "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "verified" {
		t.Fatalf("unexpected status: %s", status)
	}
	if codeRepo.markUsedCount != 1 {
		t.Fatalf("expected mark used once, got %d", codeRepo.markUsedCount)
	}
	if !u.EmailVerified || !u.IsVerified {
		t.Fatal("expected both verification flags true")
	}
}
