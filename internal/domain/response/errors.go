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
	ErrInsufficientCredits     = errors.New("insufficient credits to apply")
	ErrCreditOperationFailed   = errors.New("credit operation failed")
	ErrGeoBlocked              = errors.New("geo blocked for urgent casting")
	ErrCastingFullOrClosed     = errors.New("casting is full or closed")
	ErrRequirementsNotMet      = errors.New("model does not meet casting requirements")
)

// RequirementsError carries details about which casting requirements were not met.
// The Details map keys are field names (e.g. "height_min", "tattoos"),
// and values are human-readable explanations of the mismatch.
type RequirementsError struct {
	Details map[string]string
}

func (e *RequirementsError) Error() string {
	return ErrRequirementsNotMet.Error()
}

func (e *RequirementsError) Unwrap() error {
	return ErrRequirementsNotMet
}
