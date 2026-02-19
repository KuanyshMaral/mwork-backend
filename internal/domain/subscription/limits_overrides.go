package subscription

import (
	"time"

	"github.com/google/uuid"
)

const LimitKeyCastingResponses = "casting_responses"

type LimitOverride struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	LimitKey  string
	Delta     int
	Reason    string
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

type LimitStatus struct {
	LimitKey  string     `json:"limit_key"`
	Base      int        `json:"base"`
	Override  int        `json:"override"`
	Used      int        `json:"used"`
	Remaining int        `json:"remaining"`
	Period    string     `json:"period"`
	ResetsAt  *time.Time `json:"resets_at,omitempty"`
}

func IsAllowedLimitKey(limitKey string) bool {
	return limitKey == LimitKeyCastingResponses
}
