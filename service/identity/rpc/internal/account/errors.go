package account

import "errors"

var (
	ErrInvalid         = errors.New("invalid identity management request")
	ErrForbidden       = errors.New("identity management operation is forbidden")
	ErrNotFound        = errors.New("identity management resource not found")
	ErrConflict        = errors.New("identity management operation conflicts with existing data")
	ErrVersionConflict = errors.New("identity management version conflict")
	ErrInvalidState    = errors.New("identity management state does not allow this operation")
)
