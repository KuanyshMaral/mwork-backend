package response

import "errors"

var (
	ErrResponseNotFound        = errors.New("response not found")
	ErrAlreadyApplied          = errors.New("you have already applied to this casting")
	ErrCastingNotFound         = errors.New("casting not found")
	ErrCastingNotActive        = errors.New("casting is not active")
	ErrProfileRequired         = errors.New("you need to create a profile first")
	ErrOnlyModelsCanApply      = errors.New("only models can apply to castings")
	ErrNotCastingOwner         = errors.New("only the casting owner can manage responses")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrInsufficientCredits     = errors.New("insufficient credits to apply") // B1: Added for credit system
)
