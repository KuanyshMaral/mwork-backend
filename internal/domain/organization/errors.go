package organization

import "errors"

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrBINAlreadyExists     = errors.New("organization with this BIN/IIN already exists")
	ErrAlreadyVerified      = errors.New("organization is already verified")
	ErrCannotModifyVerified = errors.New("cannot modify verified organization")
)
