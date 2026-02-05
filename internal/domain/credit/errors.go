package credit

import "errors"

var (
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInsufficientCredits = errors.New("insufficient credits")
	ErrUserNotFound        = errors.New("user not found")
	ErrInternal            = errors.New("internal error")
)
