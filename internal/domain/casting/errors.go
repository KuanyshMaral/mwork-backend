package casting

import "errors"

var (
	ErrCastingNotFound        = errors.New("casting not found")
	ErrNotCastingOwner        = errors.New("you can only edit your own castings")
	ErrCastingNotActive       = errors.New("casting is not active")
	ErrInvalidStatus          = errors.New("invalid casting status")
	ErrCannotDeleteActive     = errors.New("cannot delete active casting, close it first")
	ErrOnlyEmployersCanCreate = errors.New("only employers can create castings")
	ErrCastingFullOrClosed    = errors.New("casting is full or closed")

	// ✅ добавь это
	ErrEmployerNotVerified = errors.New("employer account is pending verification")
)
