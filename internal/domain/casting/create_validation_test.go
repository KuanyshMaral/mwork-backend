package casting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/middleware"
)

type fakeUserRepo struct {
	user *user.User
}

func (f *fakeUserRepo) Create(ctx context.Context, u *user.User) error { return nil }
func (f *fakeUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return f.user, nil
}
func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) Update(ctx context.Context, u *user.User) error { return nil }
func (f *fakeUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeUserRepo) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	return nil
}
func (f *fakeUserRepo) UpdateVerificationFlags(ctx context.Context, id uuid.UUID, emailVerified bool, isVerified bool) error {
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
func (f *fakeUserRepo) DeductModelConnect(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeUserRepo) RefreshModelConnectsIfNeeded(ctx context.Context, id uuid.UUID, n int) error {
	return nil
}
func (f *fakeUserRepo) GetConnectsBalance(ctx context.Context, id uuid.UUID) (int, int, error) {
	return 0, 0, nil
}
func (f *fakeUserRepo) AddPurchasedModelConnects(ctx context.Context, id uuid.UUID, n int) error {
	return nil
}

type fakeCastingRepo struct{}

type trackingCastingRepo struct {
	fakeCastingRepo
	createCalled bool
	createErr    error
	casting      *Casting
}

func (f *trackingCastingRepo) Create(ctx context.Context, casting *Casting) error {
	f.createCalled = true
	return f.createErr
}

func (f *trackingCastingRepo) GetByID(ctx context.Context, id uuid.UUID) (*Casting, error) {
	if f.casting != nil && f.casting.ID == id {
		return f.casting, nil
	}
	return nil, nil
}

func (f *fakeCastingRepo) Create(ctx context.Context, casting *Casting) error { return nil }
func (f *fakeCastingRepo) GetByID(ctx context.Context, id uuid.UUID) (*Casting, error) {
	return nil, nil
}
func (f *fakeCastingRepo) Update(ctx context.Context, casting *Casting) error { return nil }
func (f *fakeCastingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	return nil
}
func (f *fakeCastingRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeCastingRepo) List(ctx context.Context, filter *Filter, sortBy SortBy, pagination *Pagination) ([]*Casting, int, error) {
	return nil, 0, nil
}
func (f *fakeCastingRepo) IncrementViewCount(ctx context.Context, id uuid.UUID) error { return nil }
func (f *fakeCastingRepo) IncrementAcceptedAndMaybeClose(ctx context.Context, id uuid.UUID) (int, Status, error) {
	return 0, "", nil
}
func (f *fakeCastingRepo) IncrementAcceptedAndMaybeCloseTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (int, Status, error) {
	return 0, "", nil
}
func (f *fakeCastingRepo) IncrementResponseCount(ctx context.Context, id uuid.UUID, delta int) error {
	return nil
}
func (f *fakeCastingRepo) IncrementResponseCountTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, delta int) error {
	return nil
}
func (f *fakeCastingRepo) ListByCreator(ctx context.Context, creatorID uuid.UUID, pagination *Pagination) ([]*Casting, int, error) {
	return nil, 0, nil
}
func (f *fakeCastingRepo) CountActiveByCreatorID(ctx context.Context, creatorID string) (int, error) {
	return 0, nil
}

func ptr[T any](v T) *T { return &v }

func TestValidateCreateCastingRequest_Table(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateCastingRequest
		wantKey string
	}{
		{name: "invalid pay range", req: CreateCastingRequest{PayMin: ptr(200.0), PayMax: ptr(100.0)}, wantKey: "pay_min"},
		{name: "invalid age range", req: CreateCastingRequest{AgeMin: ptr(30), AgeMax: ptr(20)}, wantKey: "age_min"},
		{name: "invalid height range", req: CreateCastingRequest{HeightMin: ptr(180), HeightMax: ptr(170)}, wantKey: "height_min"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCreateCastingRequest(&tc.req)
			if err == nil {
				t.Fatalf("expected validation error")
			}
			if _, ok := err[tc.wantKey]; !ok {
				t.Fatalf("expected key %q, got %+v", tc.wantKey, err)
			}
		})
	}
}

func TestServiceCreate_DateValidation(t *testing.T) {
	svc := NewService(&fakeCastingRepo{}, &fakeUserRepo{user: &user.User{Role: user.RoleEmployer, UserVerificationStatus: user.VerificationVerified}})

	_, err := svc.Create(context.Background(), uuid.New(), &CreateCastingRequest{
		Title:       "Valid title",
		Description: strings.Repeat("a", 25),
		City:        "Almaty",
		DateFrom:    ptr("10.05"),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrInvalidDateFromFormat {
		t.Fatalf("expected ErrInvalidDateFromFormat, got %v", err)
	}

	_, err = svc.Create(context.Background(), uuid.New(), &CreateCastingRequest{
		Title:       "Valid title",
		Description: strings.Repeat("a", 25),
		City:        "Almaty",
		DateFrom:    ptr(time.Now().Add(24 * time.Hour).Format(time.RFC3339)),
		DateTo:      ptr(time.Now().Format(time.RFC3339)),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestCreateRoute_InvalidDateReturns422(t *testing.T) {
	svc := NewService(&fakeCastingRepo{}, &fakeUserRepo{user: &user.User{Role: user.RoleEmployer, UserVerificationStatus: user.VerificationVerified}})
	h := NewHandler(svc, nil)

	r := chi.NewRouter()
	r.With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, uuid.New())
			ctx = context.WithValue(ctx, middleware.RequestIDKey, "test-request-id")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}).Post("/api/v1/castings", h.Create)

	body := `{"title":"Valid title","description":"` + strings.Repeat("a", 25) + `","city":"Almaty","date_from":"10.05"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/castings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestServiceCreate_EmployerNotVerifiedReturns403Error(t *testing.T) {
	repo := &trackingCastingRepo{}
	svc := NewService(repo, &fakeUserRepo{user: &user.User{Role: user.RoleEmployer, UserVerificationStatus: user.VerificationPending}})

	_, err := svc.Create(context.Background(), uuid.New(), &CreateCastingRequest{
		Title:       "Valid title",
		Description: strings.Repeat("a", 25),
		City:        "Almaty",
	})
	if err != ErrEmployerNotVerified {
		t.Fatalf("expected ErrEmployerNotVerified, got %v", err)
	}
	if repo.createCalled {
		t.Fatalf("repo.Create should not be called for unverified employer")
	}
}

func TestCreateRoute_EmployerNotVerifiedReturns403(t *testing.T) {
	repo := &trackingCastingRepo{}
	svc := NewService(repo, &fakeUserRepo{user: &user.User{Role: user.RoleEmployer, UserVerificationStatus: user.VerificationPending}})
	h := NewHandler(svc, nil)

	r := chi.NewRouter()
	r.With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), middleware.UserIDKey, uuid.New())
			ctx = context.WithValue(ctx, middleware.RequestIDKey, "test-request-id")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}).Post("/api/v1/castings", h.Create)

	body := `{"title":"Valid title","description":"` + strings.Repeat("a", 25) + `","city":"Almaty"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/castings", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
	if repo.createCalled {
		t.Fatalf("repo.Create should not be called")
	}
}

func TestServiceCreate_VerifiedEmployerProceedsToRepo(t *testing.T) {
	repo := &trackingCastingRepo{}
	svc := NewService(repo, &fakeUserRepo{user: &user.User{Role: user.RoleEmployer, UserVerificationStatus: user.VerificationVerified}})

	_, err := svc.Create(context.Background(), uuid.New(), &CreateCastingRequest{
		Title:       "Valid title",
		Description: strings.Repeat("a", 25),
		City:        "Almaty",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !repo.createCalled {
		t.Fatalf("expected repo.Create to be called")
	}
}

func TestServiceUpdate_InvalidDateFormat(t *testing.T) {
	castingID := uuid.New()
	repo := &trackingCastingRepo{casting: &Casting{ID: castingID, CreatorID: uuid.MustParse("11111111-1111-1111-1111-111111111111")}}
	svc := NewService(repo, &fakeUserRepo{user: &user.User{Role: user.RoleEmployer, UserVerificationStatus: user.VerificationVerified}})

	_, err := svc.Update(context.Background(), castingID, repo.casting.CreatorID, &UpdateCastingRequest{DateFrom: ptr("10.05")})
	if err != ErrInvalidDateFromFormat {
		t.Fatalf("expected ErrInvalidDateFromFormat, got %v", err)
	}
}

func TestServiceUpdateStatus_InvalidTransition(t *testing.T) {
	castingID := uuid.New()
	ownerID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	repo := &trackingCastingRepo{casting: &Casting{ID: castingID, CreatorID: ownerID, Status: StatusActive}}
	svc := NewService(repo, &fakeUserRepo{user: &user.User{Role: user.RoleEmployer, UserVerificationStatus: user.VerificationVerified}})

	_, err := svc.UpdateStatus(context.Background(), castingID, ownerID, StatusDraft)
	if err != ErrInvalidStatusTransition {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestList_InvalidStatusQueryReturns422(t *testing.T) {
	svc := NewService(&fakeCastingRepo{}, &fakeUserRepo{user: &user.User{Role: user.RoleEmployer, UserVerificationStatus: user.VerificationVerified}})
	h := NewHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/castings?status=deleted", nil)
	rr := httptest.NewRecorder()

	h.List(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", rr.Code, rr.Body.String())
	}
}
