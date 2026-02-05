package credit

import "errors"

var (
	// ErrInsufficientCredits is returned when user doesn't have enough credits
	ErrInsufficientCredits = errors.New("insufficient credits")

	// ErrInvalidAmount is returned when amount is <= 0
	ErrInvalidAmount = errors.New("invalid amount: must be greater than 0")

	// ErrUserNotFound is returned when user doesn't exist
	ErrUserNotFound = errors.New("user not found")

	// ErrTransactionFailed is returned when a transaction fails
	ErrTransactionFailed = errors.New("credit transaction failed")

	ErrInternal = errors.New("internal error")
)
