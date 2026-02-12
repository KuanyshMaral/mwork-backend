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
	"github.com/mwork/mwork-api/internal/pkg/password"
)

func TestLoginHandlerUnverifiedReturns403WithoutTokens(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "uv@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now(), EmailVerified: false, IsVerified: false}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)
	h := NewHandler(svc)

	body, _ := json.Marshal(LoginRequest{Email: u.Email, Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["error_code"] != "EMAIL_NOT_VERIFIED" {
		t.Fatalf("expected EMAIL_NOT_VERIFIED, got %#v", out["error_code"])
	}
	if _, ok := out["tokens"]; ok {
		t.Fatal("tokens must be absent")
	}
}

func TestLoginHandlerVerifiedReturns200WithTokens(t *testing.T) {
	hash, _ := password.Hash("password123")
	u := &user.User{ID: uuid.New(), Email: "v@example.com", PasswordHash: hash, Role: user.RoleModel, CreatedAt: time.Now(), EmailVerified: true, IsVerified: true}
	repo := &fakeUserRepo{byEmail: u, byID: u}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	svc := NewService(repo, &fakeModelProfileRepo{}, jwtService, newFakeRefreshRepo(), &fakeEmployerProfileRepo{}, nil, false, 50*time.Millisecond, nil, "pepper", false, false, nil)
	h := NewHandler(svc)

	body, _ := json.Marshal(LoginRequest{Email: u.Email, Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var out struct {
		Success bool `json:"success"`
		Data    struct {
			Tokens struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"tokens"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Data.Tokens.AccessToken == "" || out.Data.Tokens.RefreshToken == "" {
		t.Fatal("expected tokens in response")
	}
}
