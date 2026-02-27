package promotion

import "errors"

var (
	ErrPromotionNotFound  = errors.New("promotion not found")
	ErrInvalidStatus      = errors.New("invalid promotion status")
	ErrInsufficientBudget = errors.New("insufficient budget")
	ErrPaymentRequired    = errors.New("payment required to activate")
	ErrAlreadyActive      = errors.New("promotion is already active")
	ErrCannotModifyActive = errors.New("cannot modify active promotion")
	ErrPlanRequired       = errors.New("subscription plan required for promotions")
	ErrProfileNotFound    = errors.New("profile not found")
	ErrNotPromotionOwner  = errors.New("you do not own this resource")
)
