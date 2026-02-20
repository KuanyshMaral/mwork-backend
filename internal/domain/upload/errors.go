package upload

import "errors"

var (
	ErrUploadNotFound = errors.New("upload not found")
	ErrNotOwner       = errors.New("you are not the owner of this file")
	ErrFileTooLarge   = errors.New("file exceeds maximum allowed size")
	ErrInvalidMime    = errors.New("file type is not allowed")
)
