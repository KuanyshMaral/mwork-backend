package photo

import "errors"

var (
	ErrPhotoNotFound       = errors.New("photo not found")
	ErrNotPhotoOwner       = errors.New("you can only manage your own photos")
	ErrPhotoLimitReached   = errors.New("photo limit reached for your plan")
	ErrNoProfileFound      = errors.New("profile not found, create a profile first")
	ErrUploadNotVerified   = errors.New("upload could not be verified")
	ErrOnlyModelsCanUpload = errors.New("only models can upload portfolio photos")
)
