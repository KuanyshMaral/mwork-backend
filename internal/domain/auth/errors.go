package auth

import "errors"

var (
	ErrEmailAlreadyExists   = errors.New("email already registered")
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrInvalidRole          = errors.New("invalid role, must be 'model' or 'employer'")
	ErrInvalidRefreshToken  = errors.New("invalid or expired refresh token")
	ErrUserNotFound         = errors.New("user not found")
	ErrRefreshTokenRequired = errors.New("refresh token is required")
	ErrUserBanned           = errors.New("user is banned")
)
