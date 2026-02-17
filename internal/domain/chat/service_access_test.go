package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/user"
)

type testChatRepo struct {
	hasAccess bool
	room      *Room
}

func (r *testChatRepo) CreateRoom(ctx context.Context, room *Room) error             { r.room = room; return nil }
func (r *testChatRepo) GetRoomByID(ctx context.Context, id uuid.UUID) (*Room, error) { return nil, nil }
func (r *testChatRepo) GetRoomByParticipants(ctx context.Context, user1, user2 uuid.UUID) (*Room, error) {
	return nil, nil
}
func (r *testChatRepo) ListRoomsByUser(ctx context.Context, userID uuid.UUID) ([]*Room, error) {
	return nil, nil
}
func (r *testChatRepo) UpdateRoomLastMessage(ctx context.Context, roomID uuid.UUID, preview string) error {
	return nil
}
func (r *testChatRepo) DeleteRoom(ctx context.Context, id uuid.UUID) error { return nil }
func (r *testChatRepo) HasCastingResponseAccess(ctx context.Context, castingID, user1, user2 uuid.UUID) (bool, error) {
	return r.hasAccess, nil
}
func (r *testChatRepo) CreateMessage(ctx context.Context, msg *Message) error { return nil }
func (r *testChatRepo) GetMessageByID(ctx context.Context, id uuid.UUID) (*Message, error) {
	return nil, nil
}
func (r *testChatRepo) ListMessagesByRoom(ctx context.Context, roomID uuid.UUID, limit, offset int) ([]*Message, error) {
	return nil, nil
}
func (r *testChatRepo) DeleteMessage(ctx context.Context, id uuid.UUID) error { return nil }
func (r *testChatRepo) MarkMessagesAsRead(ctx context.Context, roomID, userID uuid.UUID) error {
	return nil
}
func (r *testChatRepo) CountUnreadByRoom(ctx context.Context, roomID, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (r *testChatRepo) CountUnreadByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}

type testUserRepo struct {
	users map[uuid.UUID]*user.User
}

func (r *testUserRepo) Create(ctx context.Context, u *user.User) error { return nil }
func (r *testUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	return r.users[id], nil
}
func (r *testUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return nil, nil
}
func (r *testUserRepo) Update(ctx context.Context, u *user.User) error { return nil }
func (r *testUserRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *testUserRepo) UpdateEmailVerified(ctx context.Context, id uuid.UUID, verified bool) error {
	return nil
}
func (r *testUserRepo) UpdateVerificationFlags(ctx context.Context, id uuid.UUID, emailVerified bool, isVerified bool) error {
	return nil
}
func (r *testUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return nil
}
func (r *testUserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status user.Status) error {
	return nil
}
func (r *testUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	return nil
}

type testLimitChecker struct {
	calls int
	err   error
}

func (l *testLimitChecker) CanUseChat(ctx context.Context, userID uuid.UUID) error {
	l.calls++
	return l.err
}

func TestCreateOrGetRoom_AllowsFreePlanWithResponseAccess(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	castingID := uuid.New()

	repo := &testChatRepo{hasAccess: true}
	limits := &testLimitChecker{err: errors.New("should not be used")}
	users := &testUserRepo{users: map[uuid.UUID]*user.User{
		senderID:    {ID: senderID, Role: user.RoleModel},
		recipientID: {ID: recipientID, Role: user.RoleEmployer},
	}}

	svc := NewService(repo, users, nil, nil, limits)
	room, err := svc.CreateOrGetRoom(context.Background(), senderID, &CreateRoomRequest{
		RecipientID: recipientID,
		CastingID:   &castingID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if room == nil {
		t.Fatal("expected room to be created")
	}
	if limits.calls != 0 {
		t.Fatalf("expected chat limit checker to be skipped, got %d calls", limits.calls)
	}
}

func TestCreateOrGetRoom_FallsBackToPlanLimitWithoutResponseAccess(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	castingID := uuid.New()

	repo := &testChatRepo{hasAccess: false}
	limits := &testLimitChecker{err: errors.New("chat is not available on your current plan")}
	users := &testUserRepo{users: map[uuid.UUID]*user.User{
		senderID:    {ID: senderID, Role: user.RoleModel},
		recipientID: {ID: recipientID, Role: user.RoleEmployer},
	}}

	svc := NewService(repo, users, nil, nil, limits)
	_, err := svc.CreateOrGetRoom(context.Background(), senderID, &CreateRoomRequest{
		RecipientID: recipientID,
		CastingID:   &castingID,
		Message:     time.Now().String(),
	})
	if err == nil {
		t.Fatal("expected limit error")
	}
	if limits.calls != 1 {
		t.Fatalf("expected chat limit checker to be called once, got %d", limits.calls)
	}
}
