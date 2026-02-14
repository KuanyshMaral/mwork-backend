package wallet

import "errors"

var (
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInsufficientFunds  = errors.New("insufficient wallet balance")
	ErrDuplicateReference = errors.New("duplicate reference")
	ErrReferenceConflict  = errors.New("reference conflicts with different amount")
)
