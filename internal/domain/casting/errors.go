package casting

import "errors"

var (
	ErrCastingNotFound         = errors.New("casting not found")
	ErrNotCastingOwner         = errors.New("you can only edit your own castings")
	ErrCastingNotActive        = errors.New("casting is not active")
	ErrInvalidStatus           = errors.New("invalid casting status")
	ErrInvalidStatusTransition = errors.New("invalid casting status transition")
	ErrCannotDeleteActive      = errors.New("cannot delete active casting, close it first")
	ErrOnlyEmployersCanCreate  = errors.New("only employers can create castings")
	ErrCastingFullOrClosed     = errors.New("casting is full or closed")
	ErrEmployerNotVerified     = errors.New("employer account is pending verification")

	ErrInvalidPayRange         = errors.New("invalid pay range")
	ErrInvalidDateRange        = errors.New("invalid date range")
	ErrInvalidDateFromFormat   = errors.New("invalid date_from format")
	ErrInvalidDateToFormat     = errors.New("invalid date_to format")
	ErrInvalidCreatorReference = errors.New("invalid creator_id")
	ErrDuplicateCasting        = errors.New("duplicate casting")
	ErrCastingConstraint       = errors.New("casting constraint violation")
)

// ValidationErrors carries field-level validation messages.
type ValidationErrors map[string]string

func (v ValidationErrors) Error() string {
	return "validation failed"
}
