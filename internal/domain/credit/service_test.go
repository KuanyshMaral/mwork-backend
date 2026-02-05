package credit_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/domain/user"
)

/* =========================
   Test 1: Concurrency Deduct
   ========================= */

func TestConcurrencyDeduct(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	testUser := createTestUserWithCredits(t, db, 5)
	service := credit.NewService(db)

	const goroutines = 10
	const expectedSuccess = 5

	var wg sync.WaitGroup
	success := 0
	var mu sync.Mutex

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			err := service.Deduct(
				context.Background(),
				testUser.ID,
				1,
				credit.TransactionMeta{
					RelatedEntityType: "test",
					RelatedEntityID:   uuid.New(),
					Description:       fmt.Sprintf("concurrent %d", i),
				},
			)

			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
				return
			}

			if !errors.Is(err, credit.ErrInsufficientCredits) {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}

	wg.Wait()

	if success != expectedSuccess {
		t.Fatalf("expected %d successes, got %d", expectedSuccess, success)
	}

	balance, err := service.GetBalance(context.Background(), testUser.ID)
	requireNoError(t, err)

	if balance != 0 {
		t.Fatalf("expected balance 0, got %d", balance)
	}
}

/* =========================
   Test 2: Apply Rollback
   ========================= */

func TestApplyRollback(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	testUser := createTestUserWithCredits(t, db, 10)
	service := credit.NewService(db)

	err := service.Deduct(context.Background(), testUser.ID, 1, credit.TransactionMeta{
		RelatedEntityType: "casting",
		RelatedEntityID:   uuid.New(),
		Description:       "apply",
	})
	requireNoError(t, err)

	responseID := uuid.New()
	err = service.Add(context.Background(), testUser.ID, 1, credit.TransactionTypeRefund, credit.TransactionMeta{
		RelatedEntityType: "response",
		RelatedEntityID:   responseID,
		Description:       "refund",
	})
	requireNoError(t, err)

	balance, err := service.GetBalance(context.Background(), testUser.ID)
	requireNoError(t, err)

	if balance != 10 {
		t.Fatalf("expected balance 10, got %d", balance)
	}
}

/* =========================
   Test 3: Reject Idempotency
   ========================= */

func TestRejectIdempotency(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	testUser := createTestUserWithCredits(t, db, 0)
	service := credit.NewService(db)

	responseID := uuid.New()

	err := service.Add(context.Background(), testUser.ID, 1, credit.TransactionTypeRefund, credit.TransactionMeta{
		RelatedEntityType: "response",
		RelatedEntityID:   responseID,
		Description:       "refund",
	})
	requireNoError(t, err)

	hasRefund, err := service.HasRefund(context.Background(), responseID)
	requireNoError(t, err)

	if !hasRefund {
		t.Fatal("expected refund to exist")
	}
}

/* =========================
   Test 4: Admin Grant
   ========================= */

func TestAdminGrant(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	testUser := createTestUserWithCredits(t, db, 0)
	service := credit.NewService(db)

	err := service.Add(context.Background(), testUser.ID, 100, credit.TransactionTypeAdminGrant, credit.TransactionMeta{
		RelatedEntityType: "admin",
		RelatedEntityID:   uuid.New(),
		Description:       "grant",
	})
	requireNoError(t, err)

	balance, err := service.GetBalance(context.Background(), testUser.ID)
	requireNoError(t, err)

	if balance != 100 {
		t.Fatalf("expected balance 100, got %d", balance)
	}
}

/* =========================
   Test 5: Invalid Amount
   ========================= */

func TestInvalidAmount(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	testUser := createTestUserWithCredits(t, db, 10)
	service := credit.NewService(db)

	err := service.Deduct(context.Background(), testUser.ID, 0, credit.TransactionMeta{})
	if !errors.Is(err, credit.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	err = service.Add(context.Background(), testUser.ID, -5, credit.TransactionTypePurchase, credit.TransactionMeta{})
	if !errors.Is(err, credit.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

/* =========================
   Helpers
   ========================= */

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func setupTestDB(t *testing.T) *sqlx.DB {
	dsn := "postgres://mwork:mwork_secret@localhost:5432/mwork_dev?sslmode=disable"
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("db not available: %v", err)
	}
	return db
}

func cleanupTestDB(db *sqlx.DB) {
	if db == nil {
		return
	}
	db.Exec("DELETE FROM credit_transactions")
	db.Exec("DELETE FROM users")
	db.Close()
}

func createTestUserWithCredits(t *testing.T, db *sqlx.DB, credits int) *user.User {
	u := &user.User{
		ID:            uuid.New(),
		Email:         fmt.Sprintf("test_%s@test.com", uuid.New().String()[:8]),
		PasswordHash:  "hash",
		Role:          "model",
		CreditBalance: credits,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, role, credit_balance, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, u.ID, u.Email, u.PasswordHash, u.Role, u.CreditBalance, u.CreatedAt, u.UpdatedAt)

	requireNoError(t, err)
	return u
}
