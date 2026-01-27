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
	payment, err := s.repo.GetByKaspiOrderID(ctx, kaspiOrderID)
	if err != nil || payment == nil {
		log.Warn().Str("kaspi_order_id", kaspiOrderID).Msg("Payment not found")
		return ErrPaymentNotFound
	}

	// Map string status to Status type
	var paymentStatus Status
	switch status {
	case "completed":
		paymentStatus = StatusCompleted
	case "failed":
		paymentStatus = StatusFailed
	case "pending":
		paymentStatus = StatusPending
	default:
		log.Warn().Str("status", status).Msg("Unknown status")
		return nil
	}

	// Check idempotency
	if payment.Status == paymentStatus {
		log.Info().
			Str("kaspi_order_id", kaspiOrderID).
			Str("status", string(paymentStatus)).
			Msg("Payment already in target status")
		return nil
	}

	// Update status
	if err := s.repo.UpdateStatus(ctx, payment.ID, paymentStatus); err != nil {
		log.Error().Err(err).Str("payment_id", payment.ID.String()).Msg("Failed to update payment")
		return err
	}

	// Activate subscription if payment completed
	if paymentStatus == StatusCompleted && payment.SubscriptionID.Valid {
		if err := s.subSvc.ActivateSubscription(ctx, payment.SubscriptionID.UUID); err != nil {
			log.Error().Err(err).Msg("Failed to activate subscription")
		}
	}

	log.Info().
		Str("kaspi_order_id", kaspiOrderID).
		Str("status", string(paymentStatus)).
		Msg("Payment updated successfully")

	return nil
}

// Errors
var (
	ErrPaymentNotFound = subscription.ErrPaymentFailed
)
