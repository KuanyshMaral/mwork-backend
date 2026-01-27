package user

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrEmailExists        = errors.New("email already exists") // Alias for lead module
	ErrInvalidCredentials = errors.New("invalid credentials")
)
