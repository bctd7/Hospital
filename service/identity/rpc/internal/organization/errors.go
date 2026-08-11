package organization

import "errors"

var (
	ErrInvalid              = errors.New("invalid organization request")
	ErrForbidden            = errors.New("organization operation is forbidden")
	ErrNotFound             = errors.New("organization unit not found")
	ErrConflict             = errors.New("organization operation conflicts with existing data")
	ErrVersionConflict      = errors.New("organization unit version conflict")
	ErrInvalidHierarchy     = errors.New("invalid organization hierarchy")
	ErrActiveChildren       = errors.New("organization unit has active children")
	ErrActiveDoctors        = errors.New("department has active doctors")
	ErrDirectoryUnavailable = errors.New("organization directory is unavailable")
)
