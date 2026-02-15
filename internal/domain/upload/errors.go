package upload

import "errors"

var (
	ErrUploadNotFound      = errors.New("upload not found")
	ErrNotUploadOwner      = errors.New("not upload owner")
	ErrAlreadyCommitted    = errors.New("upload already committed")
	ErrUploadExpired       = errors.New("upload has expired")
	ErrInvalidCategory     = errors.New("invalid upload category")
	ErrInvalidStatus       = errors.New("invalid upload status")
	ErrMetadataMismatch    = errors.New("uploaded file metadata mismatch")
	ErrStagingFileNotFound = errors.New("staging file not found")
)
