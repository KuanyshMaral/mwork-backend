package profile

import "errors"

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrProfileAlreadyExists = errors.New("profile already exists for this user")
	ErrNotProfileOwner      = errors.New("you can only edit your own profile")
	ErrInvalidProfileType   = errors.New("invalid profile type for this operation")
)
