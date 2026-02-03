package moderation

import "errors"

var (
	// Block errors
	ErrCannotBlockSelf = errors.New("cannot block yourself")
	ErrAlreadyBlocked  = errors.New("user already blocked")
	ErrBlockNotFound   = errors.New("block not found")
	ErrUserBlocked     = errors.New("user is blocked")

	// Report errors
	ErrCannotReportSelf    = errors.New("cannot report yourself")
	ErrReportNotFound      = errors.New("report not found")
	ErrInvalidReportReason = errors.New("invalid report reason")
	ErrInvalidReportStatus = errors.New("invalid report status")
)
