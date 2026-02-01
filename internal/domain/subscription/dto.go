package subscription

import (
	"time"

	"github.com/google/uuid"
)

// SubscribeRequest for POST /subscriptions/subscribe
type SubscribeRequest struct {
	PlanID        string `json:"plan_id" validate:"required,oneof=pro agency"`
	BillingPeriod string `json:"billing_period" validate:"required,oneof=monthly yearly"`
}

// CancelRequest for POST /subscriptions/cancel
type CancelRequest struct {
	Reason string `json:"reason,omitempty" validate:"max=500"`
}

// PlanResponse represents plan in API
type PlanResponse struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	PriceMonthly   float64  `json:"price_monthly"`
	PriceYearly    *float64 `json:"price_yearly,omitempty"`
	MaxPhotos      int      `json:"max_photos"`
	MaxResponses   int      `json:"max_responses_month"` // -1 = unlimited
	CanChat        bool     `json:"can_chat"`
	CanSeeViewers  bool     `json:"can_see_viewers"`
	PrioritySearch bool     `json:"priority_search"`
	MaxTeamMembers int      `json:"max_team_members"`
	Features       []string `json:"features"` // Human-readable feature list
}

// PlanResponseFromEntity converts plan to response
func PlanResponseFromEntity(p *Plan) *PlanResponse {
	resp := &PlanResponse{
		ID:             string(p.ID),
		Name:           p.Name,
		Description:    p.Description,
		PriceMonthly:   p.PriceMonthly,
		MaxPhotos:      p.MaxPhotos,
		MaxResponses:   p.MaxResponsesMonth,
		CanChat:        p.CanChat,
		CanSeeViewers:  p.CanSeeViewers,
		PrioritySearch: p.PrioritySearch,
		MaxTeamMembers: p.MaxTeamMembers,
		Features:       buildFeatureList(p),
	}

	if p.PriceYearly.Valid {
		resp.PriceYearly = &p.PriceYearly.Float64
	}

	return resp
}

func buildFeatureList(p *Plan) []string {
	features := []string{}

	if p.MaxPhotos > 0 {
		if p.MaxPhotos >= 100 {
			features = append(features, "Unlimited photos")
		} else {
			features = append(features, "Up to "+string(rune('0'+p.MaxPhotos))+" photos")
		}
	}

	if p.MaxResponsesMonth == -1 {
		features = append(features, "Unlimited applications")
	}

	if p.CanChat {
		features = append(features, "Chat with employers")
	}

	if p.CanSeeViewers {
		features = append(features, "See who viewed your profile")
	}

	if p.PrioritySearch {
		features = append(features, "Priority in search results")
	}

	if p.MaxTeamMembers > 0 {
		features = append(features, "Team management")
	}

	return features
}

// SubscriptionResponse represents subscription in API
type SubscriptionResponse struct {
	ID            uuid.UUID     `json:"id"`
	PlanID        string        `json:"plan_id"`
	Plan          *PlanResponse `json:"plan,omitempty"`
	Status        string        `json:"status"`
	BillingPeriod string        `json:"billing_period"`
	StartedAt     string        `json:"started_at"`
	ExpiresAt     *string       `json:"expires_at,omitempty"`
	DaysRemaining int           `json:"days_remaining"` // -1 = unlimited
	AutoRenew     bool          `json:"auto_renew"`
}

// SubscriptionResponseFromEntity converts subscription to response
func SubscriptionResponseFromEntity(s *Subscription, plan *Plan) *SubscriptionResponse {
	resp := &SubscriptionResponse{
		ID:            s.ID,
		PlanID:        string(s.PlanID),
		Status:        string(s.Status),
		BillingPeriod: string(s.BillingPeriod),
		StartedAt:     s.StartedAt.Format(time.RFC3339),
		DaysRemaining: s.DaysRemaining(),
		AutoRenew:     s.Status == StatusActive && s.PlanID != PlanFree,
	}

	if s.ExpiresAt.Valid {
		exp := s.ExpiresAt.Time.Format(time.RFC3339)
		resp.ExpiresAt = &exp
	}

	if plan != nil {
		resp.Plan = PlanResponseFromEntity(plan)
	}

	return resp
}

// LimitsResponse returns current user limits
type LimitsResponse struct {
	PlanID         string `json:"plan_id"`
	MaxPhotos      int    `json:"max_photos"`
	PhotosUsed     int    `json:"photos_used"`
	MaxResponses   int    `json:"max_responses_month"` // -1 = unlimited
	ResponsesUsed  int    `json:"responses_used"`
	CanChat        bool   `json:"can_chat"`
	CanSeeViewers  bool   `json:"can_see_viewers"`
	PrioritySearch bool   `json:"priority_search"`
}
