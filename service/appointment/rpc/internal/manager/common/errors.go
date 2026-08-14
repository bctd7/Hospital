package common

import "errors"

var (
	ErrInvalid                 = errors.New("invalid appointment request")
	ErrForbidden               = errors.New("appointment operation is forbidden")
	ErrNotFound                = errors.New("appointment data not found")
	ErrConflict                = errors.New("appointment data conflicts with existing data")
	ErrVersionConflict         = errors.New("appointment data version conflict")
	ErrInvalidState            = errors.New("appointment state does not allow the operation")
	ErrWindowConflict          = errors.New("item window is not fully contained by room window")
	ErrCapacityFull            = errors.New("appointment capacity is full")
	ErrBookingClosed           = errors.New("appointment booking window is closed")
	ErrPatientSessionOccupied  = errors.New("patient already has a pending booking in this session")
	ErrPatientWeeklyQuotaFull  = errors.New("patient weekly booking quota is exhausted")
	ErrExaminationWindowClosed = errors.New("examination can only start during the booked examination window")
	ErrNotImplemented          = errors.New("appointment operation is not implemented")
)
