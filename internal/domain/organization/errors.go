package organization

import "errors"

var (
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrBINAlreadyExists     = errors.New("organization with this BIN/IIN already exists")
	ErrAlreadyVerified      = errors.New("organization is already verified")
	ErrCannotModifyVerified = errors.New("cannot modify verified organization")
	ErrNotAuthorized        = errors.New("not authorized to perform this action")
	ErrMemberNotFound       = errors.New("member not found")
	ErrMemberAlreadyExists  = errors.New("user is already a member")
	ErrCannotRemoveOwner    = errors.New("cannot remove organization owner")
	ErrUserNotFound         = errors.New("user not found")
	ErrAlreadyFollowing     = errors.New("already following this organization")
)
