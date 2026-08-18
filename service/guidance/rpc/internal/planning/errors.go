package planning

import "errors"

var (
	ErrInvalid         = errors.New("invalid planning request")
	ErrForbidden       = errors.New("planning request forbidden")
	ErrNotFound        = errors.New("planning data not found")
	ErrNoPlan          = errors.New("no smart appointment plan available")
	ErrExistingBooking = errors.New("selected plan duplicates an existing booking")
	ErrWeeklyQuotaFull = errors.New("weekly booking quota is exhausted")
	ErrConflict        = errors.New("planning conflict")
	ErrUnavailable     = errors.New("planning dependency unavailable")
)
