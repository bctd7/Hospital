package precedence

import "errors"

var (
	ErrInvalid         = errors.New("invalid precedence rule request")
	ErrForbidden       = errors.New("precedence rule operation is forbidden")
	ErrNotFound        = errors.New("precedence rule not found")
	ErrConflict        = errors.New("precedence rule already exists")
	ErrCycle           = errors.New("precedence rule would create a cycle")
	ErrVersionConflict = errors.New("precedence rule version conflict")
)
