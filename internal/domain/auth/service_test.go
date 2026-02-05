package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
	"github.com/mwork/mwork-api/internal/pkg/jwt"
	"github.com/mwork/mwork-api/internal/pkg/photostudio"
)

type fakeUserRepo struct {
	created *user.User
}

func (f *fakeUserRepo) Create(ctx context.Context, u *user.User) error {
	f.created = u
	return nil
}

func (f *fakeUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return nil, nil
}

func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, nil
}

func (f *fakeUserRepo) Update(ctx context.Context, u *user.User) error {
	return nil
}

func (f *fakeUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (f *fakeUserRepo) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
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

func (f *fakePhotoStudioClient) SyncUser(ctx context.Context, payload photostudio.SyncUserPayload) error {
	if f.called != nil {
		f.called <- payload
	}
	return f.err
}

func TestRegisterIgnoresPhotoStudioError(t *testing.T) {
	repo := &fakeUserRepo{}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	called := make(chan photostudio.SyncUserPayload, 1)

	svc := NewService(
		repo,
		jwtService,
		nil,
		nil,
		&fakePhotoStudioClient{called: called, err: errors.New("boom")},
		true,
		50*time.Millisecond,
	)

	resp, err := svc.Register(context.Background(), &RegisterRequest{
		Email:    "user@example.com",
		Password: "password123",
		Role:     "model",
	})
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

func TestRegisterSkipsPhotoStudioWhenDisabled(t *testing.T) {
	repo := &fakeUserRepo{}
	jwtService := jwt.NewService("secret", time.Minute, time.Hour)
	called := make(chan photostudio.SyncUserPayload, 1)

	svc := NewService(
		repo,
		jwtService,
		nil,
		nil,
		&fakePhotoStudioClient{called: called},
		false,
		50*time.Millisecond,
	)

	resp, err := svc.Register(context.Background(), &RegisterRequest{
		Email:    "user2@example.com",
		Password: "password123",
		Role:     "model",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatal("expected auth response")
	}

	select {
	case <-called:
		t.Fatal("photostudio sync should not be called when disabled")
	case <-time.After(100 * time.Millisecond):
	}
}
