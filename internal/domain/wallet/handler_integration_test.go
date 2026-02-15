package wallet_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mwork/mwork-api/internal/domain/wallet"
	"github.com/mwork/mwork-api/internal/middleware"
	"github.com/mwork/mwork-api/internal/pkg/jwt"
)

type walletAPIResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Balance int64 `json:"balance"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestWalletEndpointsIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	userID := createTestUser(t, db)

	repo := wallet.NewRepository(db)
	svc := wallet.NewService(repo)
	h := wallet.NewHandler(svc)

	jwtSvc := jwt.NewService("wallet-integration-secret", time.Hour, 24*time.Hour)
	token, err := jwtSvc.GenerateAccessToken(userID, "model", false)
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}

	r := chi.NewRouter()
	r.Mount("/api/v1/demo/wallet", h.Routes(middleware.Auth(jwtSvc)))

	t.Run("GET /balance initial", func(t *testing.T) {
		resp := performWalletRequest(t, r, token, http.MethodGet, "/api/v1/demo/wallet/balance", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		body := decodeWalletResponse(t, resp)
		if !body.Success || body.Data.Balance != 0 {
			t.Fatalf("expected success=true balance=0, got success=%v balance=%d", body.Success, body.Data.Balance)
		}
	})

	t.Run("POST /topup", func(t *testing.T) {
		resp := performWalletRequest(t, r, token, http.MethodPost, "/api/v1/demo/wallet/topup", map[string]interface{}{
			"amount":       int64(1000),
			"reference_id": "topup_1",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		body := decodeWalletResponse(t, resp)
		if !body.Success || body.Data.Balance != 1000 {
			t.Fatalf("expected success=true balance=1000, got success=%v balance=%d", body.Success, body.Data.Balance)
		}
	})

	t.Run("POST /spend", func(t *testing.T) {
		resp := performWalletRequest(t, r, token, http.MethodPost, "/api/v1/demo/wallet/spend", map[string]interface{}{
			"amount":       int64(250),
			"reference_id": "feature_purchase_1",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		body := decodeWalletResponse(t, resp)
		if !body.Success || body.Data.Balance != 750 {
			t.Fatalf("expected success=true balance=750, got success=%v balance=%d", body.Success, body.Data.Balance)
		}
	})

	t.Run("POST /refund", func(t *testing.T) {
		resp := performWalletRequest(t, r, token, http.MethodPost, "/api/v1/demo/wallet/refund", map[string]interface{}{
			"amount":       int64(50),
			"reference_id": "refund_1",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		body := decodeWalletResponse(t, resp)
		if !body.Success || body.Data.Balance != 800 {
			t.Fatalf("expected success=true balance=800, got success=%v balance=%d", body.Success, body.Data.Balance)
		}
	})

	t.Run("POST /spend idempotent same reference same amount", func(t *testing.T) {
		resp := performWalletRequest(t, r, token, http.MethodPost, "/api/v1/demo/wallet/spend", map[string]interface{}{
			"amount":       int64(100),
			"reference_id": "feature_purchase_2",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("first spend expected 200, got %d", resp.Code)
		}

		resp = performWalletRequest(t, r, token, http.MethodPost, "/api/v1/demo/wallet/spend", map[string]interface{}{
			"amount":       int64(100),
			"reference_id": "feature_purchase_2",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("retry spend expected 200, got %d", resp.Code)
		}
		body := decodeWalletResponse(t, resp)
		if !body.Success || body.Data.Balance != 700 {
			t.Fatalf("expected success=true balance=700 after idempotent retry, got success=%v balance=%d", body.Success, body.Data.Balance)
		}
	})

	t.Run("POST /spend conflict same reference different amount", func(t *testing.T) {
		resp := performWalletRequest(t, r, token, http.MethodPost, "/api/v1/demo/wallet/spend", map[string]interface{}{
			"amount":       int64(120),
			"reference_id": "feature_purchase_3",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("first spend expected 200, got %d", resp.Code)
		}

		resp = performWalletRequest(t, r, token, http.MethodPost, "/api/v1/demo/wallet/spend", map[string]interface{}{
			"amount":       int64(121),
			"reference_id": "feature_purchase_3",
		})
		if resp.Code != http.StatusConflict {
			t.Fatalf("conflict retry expected 409, got %d", resp.Code)
		}
	})

	t.Run("GET /balance final", func(t *testing.T) {
		resp := performWalletRequest(t, r, token, http.MethodGet, "/api/v1/demo/wallet/balance", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		body := decodeWalletResponse(t, resp)
		if !body.Success || body.Data.Balance != 580 {
			t.Fatalf("expected final balance=580, got success=%v balance=%d", body.Success, body.Data.Balance)
		}
	})

	t.Run("JWT required", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/demo/wallet/balance", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req.WithContext(context.Background()))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 without jwt, got %d", rec.Code)
		}
	})
}

func performWalletRequest(t *testing.T, handler http.Handler, token, method, path string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload failed: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeWalletResponse(t *testing.T, rec *httptest.ResponseRecorder) walletAPIResponse {
	t.Helper()
	var out walletAPIResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode response failed: %v; body=%s", err, rec.Body.String())
	}
	return out
}
