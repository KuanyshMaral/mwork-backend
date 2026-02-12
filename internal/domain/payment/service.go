package payment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/domain/subscription"
	"github.com/mwork/mwork-api/internal/pkg/robokassa"
	"github.com/rs/zerolog/log"
)

// Service handles payment business logic
type Service struct {
	repo            Repository
	subSvc          *subscription.Service
	creditSvc       credit.Service // ✅ FIXED: Using credit.Service interface
	robokassaConfig RobokassaConfig
}

type RobokassaConfig struct {
	MerchantLogin string
	Password1     string
	Password2     string
	TestPassword1 string
	TestPassword2 string
	IsTest        bool
	HashAlgo      robokassa.HashAlgorithm
	PaymentURL    string
	ResultURL     string
	SuccessURL    string
	FailURL       string
}

// NewService creates payment service
func NewService(repo Repository, subSvc *subscription.Service) *Service {
	return &Service{
		repo:   repo,
		subSvc: subSvc,
	}
}

func (s *Service) SetRobokassaConfig(cfg RobokassaConfig) {
	s.robokassaConfig = cfg
}

type InitRobokassaPaymentRequest struct {
	UserID         uuid.UUID
	SubscriptionID uuid.UUID
	Amount         string
	Description    string
}

type InitRobokassaPaymentResponse struct {
	PaymentID  uuid.UUID `json:"payment_id"`
	InvID      int64     `json:"inv_id"`
	PaymentURL string    `json:"payment_url"`
	Status     string    `json:"status"`
}

func (s *Service) InitRobokassaPayment(ctx context.Context, req InitRobokassaPaymentRequest) (*InitRobokassaPaymentResponse, error) {
	invID := time.Now().UnixNano()
	amountRat, ok := new(big.Rat).SetString(req.Amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount")
	}
	outSum := amountRat.FloatString(2)
	shp := map[string]string{"Shp_payment_id": invIDString(invID)}
	password1 := s.robokassaConfig.Password1
	if s.robokassaConfig.IsTest && s.robokassaConfig.TestPassword1 != "" {
		password1 = s.robokassaConfig.TestPassword1
	}

	base := robokassa.BuildStartSignatureBase(s.robokassaConfig.MerchantLogin, outSum, invIDString(invID), password1, nil, shp)
	signature, err := robokassa.Sign(base, s.robokassaConfig.HashAlgo)
	if err != nil {
		return nil, err
	}

	initPayload := map[string]string{
		"MerchantLogin":  s.robokassaConfig.MerchantLogin,
		"OutSum":         outSum,
		"InvId":          invIDString(invID),
		"Description":    req.Description,
		"SignatureValue": signature,
		"ResultURL":      s.robokassaConfig.ResultURL,
		"SuccessURL":     s.robokassaConfig.SuccessURL,
		"FailURL":        s.robokassaConfig.FailURL,
		"Shp_payment_id": invIDString(invID),
	}
	if s.robokassaConfig.IsTest {
		initPayload["IsTest"] = "1"
	}

	rawInit, _ := json.Marshal(initPayload)
	payment := &Payment{
		ID:             uuid.New(),
		UserID:         req.UserID,
		SubscriptionID: uuid.NullUUID{UUID: req.SubscriptionID, Valid: true},
		Amount:         ratToFloat64(amountRat),
		Currency:       "KZT",
		Status:         StatusPending,
		Provider:       sql.NullString{String: "robokassa", Valid: true},
		ExternalID:     sql.NullString{String: invIDString(invID), Valid: true},
		RobokassaInvID: sql.NullInt64{Int64: invID, Valid: true},
		Description:    sql.NullString{String: req.Description, Valid: req.Description != ""},
		RawInitPayload: rawInit,
	}

	if err := s.repo.CreateRobokassaPending(ctx, payment); err != nil {
		return nil, err
	}

	paymentURL := s.robokassaConfig.PaymentURL + "?" + encodeQuery(initPayload)
	return &InitRobokassaPaymentResponse{PaymentID: payment.ID, InvID: invID, PaymentURL: paymentURL, Status: string(StatusPending)}, nil
}

func (s *Service) ProcessRobokassaResult(ctx context.Context, outSum, invID, signature string, shp map[string]string, rawPayload map[string]string) error {
	password2 := s.robokassaConfig.Password2
	if s.robokassaConfig.IsTest && s.robokassaConfig.TestPassword2 != "" {
		password2 = s.robokassaConfig.TestPassword2
	}

	base := robokassa.BuildResultSignatureBase(outSum, invID, password2, shp)
	expectedSignature, err := robokassa.Sign(base, s.robokassaConfig.HashAlgo)
	if err != nil {
		return err
	}
	if !robokassa.VerifySignature(expectedSignature, signature) {
		return fmt.Errorf("invalid signature")
	}

	invIDInt, err := strconv.ParseInt(invID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid inv_id: %w", err)
	}
	callbackAmount, err := robokassa.ParseAmount(outSum)
	if err != nil {
		return err
	}

	tx, err := s.repo.BeginTxx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	payment, err := s.repo.GetByRobokassaInvIDForUpdate(ctx, tx, invIDInt)
	if err != nil {
		return err
	}
	if payment == nil {
		return ErrPaymentNotFound
	}
	if payment.Status == StatusCompleted {
		return tx.Commit()
	}

	expectedAmount, err := robokassa.ParseAmount(fmt.Sprintf("%.2f", payment.Amount))
	if err != nil {
		return err
	}
	if !robokassa.AmountsEqual(expectedAmount, callbackAmount) {
		return fmt.Errorf("amount mismatch")
	}

	if err := s.repo.MarkRobokassaSucceeded(ctx, tx, payment.ID, rawPayload); err != nil {
		return err
	}

	if payment.SubscriptionID.Valid {
		if err := s.subSvc.ActivateSubscription(ctx, payment.SubscriptionID.UUID); err != nil {
			return err
		}
	}

	rawEvent, _ := json.Marshal(rawPayload)
	if err := s.repo.CreatePaymentEvent(ctx, tx, payment.ID, "robokassa.result.succeeded", rawEvent); err != nil {
		return err
	}

	return tx.Commit()
}

func invIDString(invID int64) string { return strconv.FormatInt(invID, 10) }

func encodeQuery(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values.Encode()
}

func ratToFloat64(v *big.Rat) float64 {
	out, _ := v.Float64()
	return out
}

// SetCreditService sets the credit service (optional, to avoid circular dependency)
func (s *Service) SetCreditService(creditSvc credit.Service) { // ✅ FIXED: Using credit.Service
	s.creditSvc = creditSvc
}

// CreatePayment creates a new payment for subscription
func (s *Service) CreatePayment(ctx context.Context, userID, subscriptionID uuid.UUID, amount float64, provider Provider) (*Payment, error) {
	now := time.Now()
	payment := &Payment{
		ID:             uuid.New(),
		UserID:         userID,
		SubscriptionID: uuid.NullUUID{UUID: subscriptionID, Valid: true},
		Amount:         amount,
		Currency:       "KZT",
		Status:         StatusPending,
		Provider:       sql.NullString{String: string(provider), Valid: true},
		CreatedAt:      now,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

// CreateCreditPayment creates a new payment for credit purchase
// B4: New method for credit purchases
func (s *Service) CreateCreditPayment(ctx context.Context, userID uuid.UUID, creditAmount int, totalPrice float64, provider Provider) (*Payment, error) {
	// Validate credit amount - only allow specific packages (5, 10, 25, 50)
	validPackages := map[int]bool{
		5:  true,
		10: true,
		25: true,
		50: true,
	}

	if !validPackages[creditAmount] {
		return nil, fmt.Errorf("invalid credit package: must be one of 5, 10, 25, or 50")
	}

	now := time.Now()
	payment := &Payment{
		ID:        uuid.New(),
		UserID:    userID,
		Amount:    totalPrice,
		Currency:  "KZT",
		Status:    StatusPending,
		Provider:  sql.NullString{String: string(provider), Valid: true},
		CreatedAt: now,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

// ConfirmPayment marks payment as completed and activates subscription or grants credits
// B4: Updated to handle credit purchases with idempotency
func (s *Service) ConfirmPayment(ctx context.Context, paymentID uuid.UUID) error {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil || payment == nil {
		return ErrPaymentNotFound
	}

	// B4: IDEMPOTENCY - If already paid, don't process again
	if payment.IsPaid() {
		return nil // Already processed - no duplicate credits
	}

	// Update payment status
	if err := s.repo.UpdateStatus(ctx, paymentID, StatusCompleted); err != nil {
		return err
	}

	// Activate subscription if this is a subscription payment
	if payment.SubscriptionID.Valid {
		if err := s.subSvc.ActivateSubscription(ctx, payment.SubscriptionID.UUID); err != nil {
			log.Error().Err(err).Msg("Failed to activate subscription after payment")
		}
	} else if s.creditSvc != nil {
		// B4: GRANT CREDITS FOR CREDIT PURCHASE
		creditAmount := s.determineCreditAmount(payment.Amount)
		if creditAmount > 0 {
			paymentIDStr := payment.ID.String()
			creditMeta := credit.TransactionMeta{ // ✅ FIXED: Using credit.TransactionMeta
				RelatedEntityType: "payment",
				RelatedEntityID:   payment.ID,
				Description:       fmt.Sprintf("Purchase via %s: payment %s", payment.Provider.String, payment.ID.String()),
				PaymentID:         &paymentIDStr,
			}

			// Grant credits - idempotent at payment service level
			err := s.creditSvc.Add(ctx, payment.UserID, creditAmount, credit.TransactionTypePurchase, creditMeta) // ✅ FIXED: Using credit.TransactionTypePurchase
			if err != nil {
				log.Error().Err(err).Str("payment_id", payment.ID.String()).Msg("Failed to grant credits after payment")
			}
		}
	}

	return nil
}

// determineCreditAmount maps payment amount to credit amount
// B4: Validates payment amount matches expected credit packages
func (s *Service) determineCreditAmount(amount float64) int {
	// Allowed credit packages with pricing (KZT):
	// 5 credits = 500 KZT
	// 10 credits = 900 KZT
	// 25 credits = 2000 KZT
	// 50 credits = 3500 KZT
	priceToCredits := map[float64]int{
		500.0:  5,
		900.0:  10,
		2000.0: 25,
		3500.0: 50,
	}

	if credits, ok := priceToCredits[amount]; ok {
		return credits
	}

	return 0
}

// FailPayment marks payment as failed
func (s *Service) FailPayment(ctx context.Context, paymentID uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, paymentID, StatusFailed)
}

// HandleWebhook processes payment webhook from provider
// B4: Webhook idempotency handled at ConfirmPayment level
func (s *Service) HandleWebhook(ctx context.Context, provider string, externalID string, status string) error {
	payment, err := s.repo.GetByExternalID(ctx, provider, externalID)
	if err != nil || payment == nil {
		log.Warn().Str("provider", provider).Str("external_id", externalID).Msg("Payment not found for webhook")
		return ErrPaymentNotFound
	}

	// B4: Idempotency check - skip if already processed
	if payment.IsPaid() {
		log.Info().Str("payment_id", payment.ID.String()).Msg("Payment already processed, skipping duplicate webhook")
		return nil
	}

	switch status {
	case "success", "completed", "paid":
		return s.ConfirmPayment(ctx, payment.ID)
	case "failed", "cancelled", "declined":
		return s.FailPayment(ctx, payment.ID)
	default:
		log.Warn().Str("status", status).Msg("Unknown payment status in webhook")
	}

	return nil
}

// GetPaymentHistory returns user's payment history
func (s *Service) GetPaymentHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Payment, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}

// Errors
var (
	ErrPaymentNotFound = subscription.ErrPaymentFailed
)
