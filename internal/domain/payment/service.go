package payment

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/mwork/mwork-api/internal/domain/subscription"
)

// Service handles payment business logic
type Service struct {
	repo   Repository
	subSvc *subscription.Service
}

// NewService creates payment service
func NewService(repo Repository, subSvc *subscription.Service) *Service {
	return &Service{
		repo:   repo,
		subSvc: subSvc,
	}
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

// ConfirmPayment marks payment as completed and activates subscription
func (s *Service) ConfirmPayment(ctx context.Context, paymentID uuid.UUID) error {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil || payment == nil {
		return ErrPaymentNotFound
	}

	if payment.IsPaid() {
		return nil // Already processed
	}

	// Update payment status
	if err := s.repo.UpdateStatus(ctx, paymentID, StatusCompleted); err != nil {
		return err
	}

	// Activate subscription
	if payment.SubscriptionID.Valid {
		if err := s.subSvc.ActivateSubscription(ctx, payment.SubscriptionID.UUID); err != nil {
			log.Error().Err(err).Msg("Failed to activate subscription after payment")
		}
	}

	return nil
}

// FailPayment marks payment as failed
func (s *Service) FailPayment(ctx context.Context, paymentID uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, paymentID, StatusFailed)
}

// HandleWebhook processes payment webhook from provider
func (s *Service) HandleWebhook(ctx context.Context, provider string, externalID string, status string) error {
	payment, err := s.repo.GetByExternalID(ctx, provider, externalID)
	if err != nil || payment == nil {
		log.Warn().Str("provider", provider).Str("external_id", externalID).Msg("Payment not found for webhook")
		return ErrPaymentNotFound
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
func (s *Service) UpdatePaymentByKaspiOrderID(ctx context.Context, kaspiOrderID string, status string) error {
	// For completed payments, use the specialized ConfirmPayment method
	if status == "completed" {
		err := s.repo.ConfirmPayment(ctx, kaspiOrderID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Warn().Str("kaspi_order_id", kaspiOrderID).Msg("Payment not found or already processed")
				return ErrPaymentNotFound
			}
			return err
		}

		// Get the payment to activate subscription
		// Note: We need to get by Kaspi order ID - using HandleWebhook logic
		payment, err := s.repo.GetByExternalID(ctx, "kaspi", kaspiOrderID)
		if err == nil && payment != nil && payment.SubscriptionID.Valid {
			if err := s.subSvc.ActivateSubscription(ctx, payment.SubscriptionID.UUID); err != nil {
				log.Error().Err(err).Msg("Failed to activate subscription after payment")
			}
		}

		return nil
	}

	// For other statuses, use the generic HandleWebhook
	return s.HandleWebhook(ctx, "kaspi", kaspiOrderID, status)
}

// Errors
var (
	ErrPaymentNotFound = subscription.ErrPaymentFailed
)
