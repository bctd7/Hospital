package patient

import (
	"context"
	"time"

	"hospital/service/appointment/rpc/internal/manager/common"
)

type (
	Cache                  = common.Cache
	BookingTxStore         = common.BookingTxStore
	Booking                = common.Booking
	BookingOption          = common.BookingOption
	BookingSelection       = common.BookingSelection
	DateCapacity           = common.DateCapacity
	BookingListFilter      = common.BookingListFilter
	BookingOperation       = common.BookingOperation
	BookingOperationChange = common.BookingOperationChange
	BookingStatus          = common.BookingStatus
	Session                = common.Session
	Page[T any]            = common.Page[T]
	flightGroup            = common.FlightGroup
)

const (
	StatusActive           = common.StatusActive
	BookingStatusConfirmed = common.BookingStatusConfirmed
)

var (
	ErrForbidden              = common.ErrForbidden
	ErrNotFound               = common.ErrNotFound
	ErrInvalid                = common.ErrInvalid
	ErrConflict               = common.ErrConflict
	ErrInvalidState           = common.ErrInvalidState
	ErrCapacityFull           = common.ErrCapacityFull
	ErrBookingClosed          = common.ErrBookingClosed
	ErrPatientSessionOccupied = common.ErrPatientSessionOccupied
)

func loadCached[T any](ctx context.Context, cache Cache, flights *flightGroup, key string, ttl time.Duration, loader func() (T, bool, error)) (T, bool, error) {
	return common.LoadCached(ctx, cache, flights, key, ttl, loader)
}
