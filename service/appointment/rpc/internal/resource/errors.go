package resource

import "errors"

var (
	ErrInvalid         = errors.New("invalid appointment resource request")
	ErrForbidden       = errors.New("appointment resource operation is forbidden")
	ErrNotFound        = errors.New("appointment resource not found")
	ErrConflict        = errors.New("appointment resource conflicts with existing data")
	ErrVersionConflict = errors.New("appointment resource version conflict")
	ErrInvalidState    = errors.New("appointment resource state does not allow the operation")
	ErrWindowConflict  = errors.New("item window is not fully contained by room window")
)
