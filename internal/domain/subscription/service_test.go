package subscription

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type repoStub struct {
	plan     *Plan
	audience Audience
	delta    int
}

func (r *repoStub) GetPlanByID(context.Context, PlanID) (*Plan, error) { return r.plan, nil }
func (r *repoStub) ListPlans(context.Context) ([]*Plan, error)         { return nil, nil }
func (r *repoStub) Create(context.Context, *Subscription) error        { return nil }
func (r *repoStub) GetByID(context.Context, uuid.UUID) (*Subscription, error) {
	return nil, nil
}
func (r *repoStub) GetActiveByUserID(context.Context, uuid.UUID) (*Subscription, error) {
	return nil, nil
}
func (r *repoStub) Update(context.Context, *Subscription) error              { return nil }
func (r *repoStub) Cancel(context.Context, uuid.UUID, string) error          { return nil }
func (r *repoStub) ExpireOldSubscriptions(context.Context) (int, error)      { return 0, nil }
func (r *repoStub) GetUserRole(context.Context, uuid.UUID) (Audience, error) { return r.audience, nil }
func (r *repoStub) GetLimitOverrideTotal(context.Context, uuid.UUID, string) (int, error) {
	return r.delta, nil
}
func (r *repoStub) CreateLimitOverride(_ context.Context, o *LimitOverride) error {
	r.delta += o.Delta
	return nil
}
func (r *repoStub) GetAllLimitOverrides(context.Context, uuid.UUID) (map[string]int, error) {
	return map[string]int{}, nil
}

type respRepoStub struct{ used int }

func (r *respRepoStub) CountMonthlyByUserID(context.Context, uuid.UUID) (int, error) {
	return r.used, nil
}

type photoRepoStub struct{}

func (p *photoRepoStub) CountByProfileID(context.Context, uuid.UUID) (int, error) { return 0, nil }

type castingRepoStub struct{}

func (c *castingRepoStub) CountActiveByCreatorID(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

type profileRepoStub struct{}

func (p *profileRepoStub) GetByUserID(context.Context, uuid.UUID) (*Profile, error) {
	return &Profile{ID: uuid.New()}, nil
}

func TestSubscribeAudienceMismatch(t *testing.T) {
	svc := NewService(&repoStub{plan: &Plan{ID: PlanAgency, Audience: AudienceEmployer}, audience: AudienceModel}, &photoRepoStub{}, &respRepoStub{}, &castingRepoStub{}, &profileRepoStub{})
	_, err := svc.Subscribe(context.Background(), uuid.New(), &SubscribeRequest{PlanID: "agency", BillingPeriod: "monthly"})
	if err != ErrPlanAudienceMismatch {
		t.Fatalf("expected ErrPlanAudienceMismatch, got %v", err)
	}
}

func TestAdjustLimitUpdatesRemaining(t *testing.T) {
	svc := NewService(&repoStub{plan: &Plan{ID: PlanPro, Audience: AudienceModel, Consumables: ConsumablesConfig{ResponseConnects: 20}}, audience: AudienceModel}, &photoRepoStub{}, &respRepoStub{used: 7}, &castingRepoStub{}, &profileRepoStub{})
	status, err := svc.AdjustLimit(context.Background(), uuid.New(), uuid.New(), LimitKeyCastingResponses, 10, "bonus")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if status.Remaining != 23 {
		t.Fatalf("expected remaining 23, got %d", status.Remaining)
	}
}
