package patient

import (
	"context"
	"time"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager"
)

type (
	Cache                  = appointmentmanager.Cache
	ProjectStore           = appointmentmanager.ProjectStore
	ProjectListFilter      = appointmentmanager.ProjectListFilter
	ExaminationItem        = appointmentmanager.ExaminationItem
	RoomScheduleStore      = appointmentmanager.RoomScheduleStore
	BookingStore           = appointmentmanager.BookingStore
	BookingTxStore         = appointmentmanager.BookingTxStore
	Booking                = appointmentmanager.Booking
	BookingOption          = appointmentmanager.BookingOption
	BookingSelection       = appointmentmanager.BookingSelection
	DateCapacity           = appointmentmanager.DateCapacity
	BookingListFilter      = appointmentmanager.BookingListFilter
	BookingOperation       = appointmentmanager.BookingOperation
	BookingOperationChange = appointmentmanager.BookingOperationChange
	BookingStatus          = appointmentmanager.BookingStatus
	Session                = appointmentmanager.Session
	Page[T any]            = appointmentmanager.Page[T]
	RoomItem               = appointmentmanager.RoomItem
	ItemWeeklyWindow       = appointmentmanager.ItemWeeklyWindow
	ItemSummary            = appointmentmanager.ItemSummary
	flightGroup            = appointmentmanager.FlightGroup
)

const (
	StatusActive           = appointmentmanager.StatusActive
	BookingStatusConfirmed = appointmentmanager.BookingStatusConfirmed
	BookingStatusCheckedIn = appointmentmanager.BookingStatusCheckedIn
	hotReadCacheTTL        = appointmentmanager.HotReadCacheTTL
	queryCacheTTL          = appointmentmanager.QueryCacheTTL
)

var (
	ErrForbidden              = appointmentmanager.ErrForbidden
	ErrNotFound               = appointmentmanager.ErrNotFound
	ErrInvalid                = appointmentmanager.ErrInvalid
	ErrConflict               = appointmentmanager.ErrConflict
	ErrInvalidState           = appointmentmanager.ErrInvalidState
	ErrCapacityFull           = appointmentmanager.ErrCapacityFull
	ErrBookingClosed          = appointmentmanager.ErrBookingClosed
	ErrPatientSessionOccupied = appointmentmanager.ErrPatientSessionOccupied
)

func loadCached[T any](ctx context.Context, cache Cache, flights *flightGroup, key string, ttl time.Duration, loader func() (T, bool, error)) (T, bool, error) {
	return appointmentmanager.LoadCached(ctx, cache, flights, key, ttl, loader)
}
