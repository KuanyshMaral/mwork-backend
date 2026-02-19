package subscription

import "errors"

var (
	ErrPlanNotFound         = errors.New("plan not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrAlreadySubscribed    = errors.New("user already has active subscription")
	ErrCannotCancelFree     = errors.New("cannot cancel free subscription")
	ErrInvalidBillingPeriod = errors.New("invalid billing period")
	ErrPaymentRequired      = errors.New("payment required for this plan")
	ErrPaymentFailed        = errors.New("payment failed")
	ErrPlanAudienceMismatch = errors.New("plan audience does not match user role")
	ErrInvalidLimitKey      = errors.New("invalid limit key")
	ErrLimitWouldBeNegative = errors.New("limit would become negative")
)
