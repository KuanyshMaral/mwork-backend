package wallet_test

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

	"github.com/mwork/mwork-api/internal/domain/wallet"
)

func TestWalletConcurrentSpend(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	userID := createTestUser(t, db)
	repo := wallet.NewRepository(db)
	svc := wallet.NewService(repo)

	if err := svc.TopUp(context.Background(), userID, 5, "seed-1"); err != nil {
		t.Fatalf("topup failed: %v", err)
	}

	const workers = 10
	var wg sync.WaitGroup
	success := 0
	var mu sync.Mutex

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := svc.Spend(context.Background(), userID, 1, fmt.Sprintf("spend-%d", i))
			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
				return
			}
			if !errors.Is(err, wallet.ErrInsufficientFunds) {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()

	if success != 5 {
		t.Fatalf("expected 5 successful spends, got %d", success)
	}

	balance, err := svc.GetBalance(context.Background(), userID)
	if err != nil {
		t.Fatalf("get balance failed: %v", err)
	}
	if balance != 0 {
		t.Fatalf("expected balance 0, got %d", balance)
	}
}

func TestWalletSpendIdempotency(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	userID := createTestUser(t, db)
	repo := wallet.NewRepository(db)
	svc := wallet.NewService(repo)

	if err := svc.TopUp(context.Background(), userID, 100, "seed-2"); err != nil {
		t.Fatalf("topup failed: %v", err)
	}

	if err := svc.Spend(context.Background(), userID, 40, "feature_purchase_123"); err != nil {
		t.Fatalf("first spend failed: %v", err)
	}
	if err := svc.Spend(context.Background(), userID, 40, "feature_purchase_123"); err != nil {
		t.Fatalf("idempotent retry failed: %v", err)
	}

	balance, err := svc.GetBalance(context.Background(), userID)
	if err != nil {
		t.Fatalf("get balance failed: %v", err)
	}
	if balance != 60 {
		t.Fatalf("expected balance 60 after idempotent spend retry, got %d", balance)
	}
}

func TestWalletReferenceConflict(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	userID := createTestUser(t, db)
	repo := wallet.NewRepository(db)
	svc := wallet.NewService(repo)

	if err := svc.TopUp(context.Background(), userID, 100, "seed-3"); err != nil {
		t.Fatalf("topup failed: %v", err)
	}

	if err := svc.Spend(context.Background(), userID, 40, "feature_purchase_456"); err != nil {
		t.Fatalf("first spend failed: %v", err)
	}

	err := svc.Spend(context.Background(), userID, 41, "feature_purchase_456")
	if !errors.Is(err, wallet.ErrReferenceConflict) {
		t.Fatalf("expected ErrReferenceConflict, got %v", err)
	}
}

func TestWalletInvalidAmount(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	userID := createTestUser(t, db)
	repo := wallet.NewRepository(db)
	svc := wallet.NewService(repo)

	if err := svc.TopUp(context.Background(), userID, 0, "x"); !errors.Is(err, wallet.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}

	if err := svc.Spend(context.Background(), userID, 1, ""); !errors.Is(err, wallet.ErrInvalidAmount) {
		t.Fatalf("expected ErrInvalidAmount for empty spend reference, got %v", err)
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
	db.Exec("DELETE FROM wallet_transactions")
	db.Exec("DELETE FROM user_wallets")
	db.Exec("DELETE FROM users")
	db.Close()
}

func createTestUser(t *testing.T, db *sqlx.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, fmt.Sprintf("wallet_%s@test.com", id.String()[:8]), "hash", "model", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	return id
}
