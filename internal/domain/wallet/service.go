package wallet

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetBalance(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.GetBalance(ctx, userID)
}

func (s *Service) TopUp(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if err := s.repo.TopUp(ctx, userID, amount, referenceID); err != nil {
		return err
	}
	log.Info().Str("user_id", userID.String()).Int64("amount", amount).Str("reference_id", referenceID).Msg("wallet topup applied")
	return nil
}

func (s *Service) Spend(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	if amount <= 0 || referenceID == "" {
		return ErrInvalidAmount
	}
	if err := s.repo.Spend(ctx, userID, amount, referenceID); err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			return ErrInsufficientFunds
		}
		return err
	}
	log.Info().Str("user_id", userID.String()).Int64("amount", amount).Str("reference_id", referenceID).Msg("wallet payment applied")
	return nil
}

func (s *Service) Refund(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	if amount <= 0 || referenceID == "" {
		return ErrInvalidAmount
	}
	if err := s.repo.Refund(ctx, userID, amount, referenceID); err != nil {
		return err
	}
	log.Info().Str("user_id", userID.String()).Int64("amount", amount).Str("reference_id", referenceID).Msg("wallet refund applied")
	return nil
}
