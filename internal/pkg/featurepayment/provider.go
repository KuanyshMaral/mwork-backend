package featurepayment

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"

	"github.com/mwork/mwork-api/internal/domain/credit"
	"github.com/mwork/mwork-api/internal/domain/wallet"
)

const (
	ModeDemo = "demo"
	ModeReal = "real"
)

type PaymentProvider interface {
	Charge(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error
	Refund(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error
}

type DemoPaymentProvider struct {
	walletSvc *wallet.Service
}

func NewDemoPaymentProvider(walletSvc *wallet.Service) *DemoPaymentProvider {
	return &DemoPaymentProvider{walletSvc: walletSvc}
}

func (p *DemoPaymentProvider) Charge(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	return p.walletSvc.Spend(ctx, userID, amount, referenceID)
}

func (p *DemoPaymentProvider) Refund(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	return p.walletSvc.Refund(ctx, userID, amount, referenceID)
}

type RealPaymentProvider struct {
	creditSvc credit.Service
}

func NewRealPaymentProvider(creditSvc credit.Service) *RealPaymentProvider {
	return &RealPaymentProvider{creditSvc: creditSvc}
}

func (p *RealPaymentProvider) Charge(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	if amount > math.MaxInt {
		return fmt.Errorf("amount too large")
	}

	meta := credit.TransactionMeta{
		RelatedEntityType: "feature",
		Description:       "charged via real payment provider",
	}
	return p.creditSvc.Deduct(ctx, userID, int(amount), meta)
}

func (p *RealPaymentProvider) Refund(ctx context.Context, userID uuid.UUID, amount int64, referenceID string) error {
	if amount > math.MaxInt {
		return fmt.Errorf("amount too large")
	}

	meta := credit.TransactionMeta{
		RelatedEntityType: "feature",
		Description:       "refund via real payment provider",
	}
	return p.creditSvc.Add(ctx, userID, int(amount), credit.TransactionTypeRefund, meta)
}

func NewPaymentProvider(mode string, walletSvc *wallet.Service, creditSvc credit.Service) (PaymentProvider, error) {
	switch mode {
	case "", ModeReal:
		if creditSvc == nil {
			return nil, fmt.Errorf("credit service is required in real payment mode")
		}
		return NewRealPaymentProvider(creditSvc), nil
	case ModeDemo:
		if walletSvc == nil {
			return nil, fmt.Errorf("wallet service is required in demo payment mode")
		}
		return NewDemoPaymentProvider(walletSvc), nil
	default:
		return nil, fmt.Errorf("unsupported payment mode: %s", mode)
	}
}
