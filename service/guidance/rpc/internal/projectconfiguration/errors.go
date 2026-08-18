package projectconfiguration

import "errors"

var (
	ErrInvalid         = errors.New("invalid examination item configuration")
	ErrForbidden       = errors.New("examination item configuration is forbidden")
	ErrNotFound        = errors.New("examination item configuration not found")
	ErrConflict        = errors.New("examination item configuration conflicts with existing data")
	ErrCycle           = errors.New("examination item configuration creates a precedence cycle")
	ErrVersionConflict = errors.New("examination item configuration version conflict")
)
