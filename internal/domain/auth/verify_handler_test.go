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
	"github.com/mwork/mwork-api/internal/pkg/jwt"
)

func TestPublicVerifyConfirmByEmailAndCode(t *testing.T) {
	u := &user.User{ID: uuid.New(), Email: "public@example.com", Role: user.RoleModel, CreatedAt: time.Now()}
	repo := &fakeUserRepo{byID: u, byEmail: u}
	codeRepo := &fakeVerificationCodeRepo{rec: &VerificationCodeRecord{UserID: u.ID, Attempts: 0, ExpiresAt: time.Now().Add(time.Minute)}}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, codeRepo, "pepper", false, false, nil)
	codeRepo.rec.CodeHash = svc.hashVerificationCode("123456")

	h := NewHandler(svc)
	router := h.Routes(func(next http.Handler) http.Handler { return next })

	body, _ := json.Marshal(map[string]string{"email": u.Email, "code": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/verify/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !u.EmailVerified || !u.IsVerified {
		t.Fatal("expected user verification flags set")
	}

	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data := out["data"].(map[string]interface{})
	if data["status"] != "verified" {
		t.Fatalf("expected verified status, got %#v", data["status"])
	}
}

func TestPublicVerifyConfirmUnknownEmailReturnsInvalidCodeWithoutAttempts(t *testing.T) {
	repo := &fakeUserRepo{}
	codeRepo := &fakeVerificationCodeRepo{rec: &VerificationCodeRecord{UserID: uuid.New(), Attempts: 0, ExpiresAt: time.Now().Add(time.Minute)}}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, codeRepo, "pepper", false, false, nil)

	h := NewHandler(svc)
	router := h.Routes(func(next http.Handler) http.Handler { return next })

	body, _ := json.Marshal(map[string]string{"email": "missing@example.com", "code": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/verify/confirm", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	errObj, _ := out["error"].(map[string]interface{})
	if errObj["code"] != "INVALID_CODE" {
		t.Fatalf("expected INVALID_CODE, got %#v", errObj["code"])
	}
	if codeRepo.incrementCount != 0 {
		t.Fatalf("expected attempts to stay unchanged for unknown email, got incrementCount=%d", codeRepo.incrementCount)
	}
}
