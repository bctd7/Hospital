package catalog

import "errors"

var (
	ErrInvalid         = errors.New("invalid examination catalog request")
	ErrForbidden       = errors.New("examination catalog operation is forbidden")
	ErrNotFound        = errors.New("examination item not found")
	ErrConflict        = errors.New("examination item conflicts with existing data")
	ErrVersionConflict = errors.New("examination item version conflict")
	ErrInvalidState    = errors.New("examination item state does not allow the operation")
	ErrNotImplemented  = errors.New("examination catalog operation is not implemented")
)
