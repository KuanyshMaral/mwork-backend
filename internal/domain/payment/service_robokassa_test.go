package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/mwork/mwork-api/internal/pkg/robokassa"
)

type captureRepo struct {
	created *Payment
}

func (r *captureRepo) Create(ctx context.Context, p *Payment) error                { return nil }
func (r *captureRepo) GetByID(ctx context.Context, id uuid.UUID) (*Payment, error) { return nil, nil }
func (r *captureRepo) GetByExternalID(ctx context.Context, provider, externalID string) (*Payment, error) {
	return nil, nil
}
func (r *captureRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status Status) error {
	return nil
}
func (r *captureRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Payment, error) {
	return nil, nil
}
func (r *captureRepo) GetByPromotionID(ctx context.Context, promotionID string) (*Payment, error) {
	return nil, nil
}
func (r *captureRepo) CreateRobokassaPending(ctx context.Context, payment *Payment) error {
	r.created = payment
	return nil
}
func (r *captureRepo) GetByRobokassaInvIDForUpdate(ctx context.Context, tx *sqlx.Tx, invID int64) (*Payment, error) {
	return nil, nil
}
func (r *captureRepo) GetByInvIDForUpdate(ctx context.Context, tx *sqlx.Tx, invID string) (*Payment, error) {
	return nil, nil
}
func (r *captureRepo) MarkRobokassaSucceeded(ctx context.Context, tx *sqlx.Tx, paymentID uuid.UUID, callbackPayload map[string]string) error {
	return nil
}
func (r *captureRepo) CreatePaymentEvent(ctx context.Context, tx *sqlx.Tx, paymentID uuid.UUID, eventType string, payload any) error {
	return nil
}
func (r *captureRepo) BeginTxx(ctx context.Context) (*sqlx.Tx, error)        { return nil, nil }
func (r *captureRepo) NextRobokassaInvID(ctx context.Context) (int64, error) { return 1001, nil }

func TestIsTestCallback(t *testing.T) {
	if !isTestCallback(map[string]string{"IsTest": "1"}) {
		t.Fatal("expected true for IsTest=1")
	}
	if isTestCallback(map[string]string{"IsTest": "0"}) {
		t.Fatal("expected false for IsTest=0")
	}
}

func TestNormalizeAmount_Comma(t *testing.T) {
	a, err := normalizeAmount("100,50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.FloatString(2) != "100.50" {
		t.Fatalf("unexpected normalized amount: %s", a.FloatString(2))
	}
}

func TestNormalizeAmount_NonPositiveRejected(t *testing.T) {
	if _, err := normalizeAmount("0"); err == nil {
		t.Fatal("expected error for zero amount")
	}
	if _, err := normalizeAmount("-10"); err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestSetRobokassaConfig_InvalidHashAlgorithm(t *testing.T) {
	svc := NewService(nil, nil)
	svc.SetRobokassaConfig(RobokassaConfig{
		MerchantLogin: "merchant",
		Password1:     "p1",
		Password2:     "p2",
		HashAlgo:      "BAD",
	})
	if svc.robokassaErr == nil {
		t.Fatal("expected configuration error for invalid hash algorithm")
	}
}

func TestValidateRobokassaReplayProtection(t *testing.T) {
	payment := &Payment{
		UserID:         uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		InvID:          sql.NullString{String: "1001", Valid: true},
		RawInitPayload: []byte(`{"Shp_user":"11111111-1111-1111-1111-111111111111","Shp_nonce":"11111111-1111-1111-1111-111111111111-1001"}`),
	}
	svc := NewService(nil, nil)

	ok := map[string]string{
		"shp_user":  "11111111-1111-1111-1111-111111111111",
		"SHP_NONCE": "11111111-1111-1111-1111-111111111111-1001",
	}
	if err := svc.validateRobokassaReplayProtection(payment, ok); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bad := map[string]string{
		"Shp_user":  "11111111-1111-1111-1111-111111111111",
		"Shp_nonce": "wrong",
	}
	if err := svc.validateRobokassaReplayProtection(payment, bad); err == nil {
		t.Fatal("expected replay protection error")
	}
}

func TestValidateRobokassaReplayProtection_LegacyPaymentWithoutShp(t *testing.T) {
	payment := &Payment{UserID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), InvID: sql.NullString{String: "1001", Valid: true}}
	svc := NewService(nil, nil)
	if err := svc.validateRobokassaReplayProtection(payment, nil); err != nil {
		t.Fatalf("legacy payment should not fail replay check, got: %v", err)
	}
}

func TestCreateResponsePayment_PersistsInitPayloadWithShp(t *testing.T) {
	repo := &captureRepo{}
	svc := NewService(repo, nil)
	svc.SetRobokassaConfig(RobokassaConfig{MerchantLogin: "merchant", Password1: "p1", Password2: "p2", HashAlgo: "sha256"})

	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	_, err := svc.CreateResponsePayment(context.Background(), userID, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected pending payment to be created")
	}

	var payload map[string]string
	if err := json.Unmarshal(repo.created.RawInitPayload, &payload); err != nil {
		t.Fatalf("invalid raw init payload: %v", err)
	}
	if payload["Shp_user"] != userID.String() {
		t.Fatalf("unexpected Shp_user: %q", payload["Shp_user"])
	}
	if !strings.HasPrefix(payload["Shp_nonce"], userID.String()+"-1001-") {
		t.Fatalf("unexpected Shp_nonce: %q", payload["Shp_nonce"])
	}
}

func TestVerifyRobokassaSuccessRedirect_NormalizesAmountForSignature(t *testing.T) {
	svc := NewService(nil, nil)
	svc.SetRobokassaConfig(RobokassaConfig{MerchantLogin: "merchant", Password1: "p1", Password2: "p2", HashAlgo: "sha256"})

	invID := "1001"
	shp := map[string]string{"Shp_user": "user-1", "Shp_nonce": "nonce-1"}
	base := robokassa.BuildSuccessSignatureBase("100.50", invID, "p1", shp)
	sig, err := robokassa.Sign(base, robokassa.HashSHA256)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	if err := svc.VerifyRobokassaSuccessRedirect("100,50", invID, sig, shp); err != nil {
		t.Fatalf("expected normalized amount to validate signature: %v", err)
	}
}
