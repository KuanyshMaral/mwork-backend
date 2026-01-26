package lead

import "errors"

var (
	ErrLeadNotFound      = errors.New("lead not found")
	ErrLeadAlreadyExists = errors.New("lead with this email already exists")
	ErrAlreadyConverted  = errors.New("lead is already converted")
	ErrCannotConvert     = errors.New("lead cannot be converted in current status")
)
