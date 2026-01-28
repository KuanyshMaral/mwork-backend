package experience

import "errors"

var (
	// ErrExperienceNotFound is returned when experience is not found
	ErrExperienceNotFound = errors.New("experience not found")

	// ErrNotExperienceOwner is returned when user is not the owner of experience
	ErrNotExperienceOwner = errors.New("not experience owner")
)
