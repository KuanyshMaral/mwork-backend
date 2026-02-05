package payment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/domain/subscription"
)

// Service handles payment business logic
type Service struct {
	repo      Repository
	subSvc    *subscription.Service
	creditSvc credit.Service // ✅ FIXED: Using credit.Service interface
}

// NewService creates payment service
func NewService(repo Repository, subSvc *subscription.Service) *Service {
	return &Service{
		repo:   repo,
		subSvc: subSvc,
	}
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

// UpdatePaymentByKaspiOrderID updates payment status by Kaspi order ID
// B4: Kaspi webhook with idempotency protection
func (s *Service) UpdatePaymentByKaspiOrderID(ctx context.Context, kaspiOrderID string, status string) error {
	// For completed payments
	if status == "completed" {
		// Get payment first to check if already processed
		payment, err := s.repo.GetByExternalID(ctx, "kaspi", kaspiOrderID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Warn().Str("kaspi_order_id", kaspiOrderID).Msg("Payment not found")
				return ErrPaymentNotFound
			}
			return err
		}

		// B4: IDEMPOTENCY - Don't process if already paid
		if payment.IsPaid() {
			log.Info().Str("kaspi_order_id", kaspiOrderID).Msg("Payment already processed")
			return nil
		}

		// Update payment status
		err = s.repo.ConfirmPayment(ctx, kaspiOrderID)
		if err != nil {
			return err
		}

		// Process subscription or credits
		if payment.SubscriptionID.Valid {
			if err := s.subSvc.ActivateSubscription(ctx, payment.SubscriptionID.UUID); err != nil {
				log.Error().Err(err).Msg("Failed to activate subscription after payment")
			}
		} else if s.creditSvc != nil {
			// B4: Grant credits for credit purchase
			creditAmount := s.determineCreditAmount(payment.Amount)
			if creditAmount > 0 {
				paymentIDStr := payment.ID.String()
				creditMeta := credit.TransactionMeta{ // ✅ FIXED: Using credit.TransactionMeta
					RelatedEntityType: "payment",
					RelatedEntityID:   payment.ID,
					Description:       fmt.Sprintf("Purchase via Kaspi: order %s", kaspiOrderID),
					PaymentID:         &paymentIDStr,
				}

				_ = s.creditSvc.Add(ctx, payment.UserID, creditAmount, credit.TransactionTypePurchase, creditMeta) // ✅ FIXED: Using credit.TransactionTypePurchase
			}
		}

		return nil
	}

	// For other statuses
	return s.HandleWebhook(ctx, "kaspi", kaspiOrderID, status)
}

// Errors
var (
	ErrPaymentNotFound = subscription.ErrPaymentFailed
)
