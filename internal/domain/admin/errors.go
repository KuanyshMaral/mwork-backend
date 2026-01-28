package admin

import "errors"

var (
	ErrAdminNotFound       = errors.New("admin not found")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrAdminInactive       = errors.New("admin account is inactive")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrCannotManageRole    = errors.New("cannot manage admin with equal or higher role")
	ErrCannotDeleteSelf    = errors.New("cannot delete your own account")
	ErrEmailTaken          = errors.New("email already in use")
	ErrFeatureFlagNotFound = errors.New("feature flag not found")
	ErrReportNotFound      = errors.New("report not found")
)
