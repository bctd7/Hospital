package manager

import (
	"context"
	"time"
)

type BookingStore interface {
	ListBookingOptions(ctx context.Context, itemID string, fromDate, throughDate time.Time) ([]BookingOption, error)
	GetBooking(ctx context.Context, bookingID string) (Booking, error)
	ListBookings(ctx context.Context, filter BookingListFilter) ([]Booking, int64, error)
	WithinBookingTransaction(ctx context.Context, fn func(BookingTxStore) error) error
}

type BookingTxStore interface {
	FindBookingOperation(ctx context.Context, operationID string) (BookingOperation, bool, error)
	RecordBookingOperation(ctx context.Context, change BookingOperationChange) error
	ListBookingsForUpdate(ctx context.Context, filter BookingListFilter) ([]Booking, error)
	LockBookingSelection(ctx context.Context, itemID, roomID string, serviceDate time.Time, session Session) (BookingSelection, error)
	FindDateCapacityForUpdate(ctx context.Context, roomID string, serviceDate time.Time, session Session) (DateCapacity, bool, error)
	CreateDateCapacity(ctx context.Context, capacity DateCapacity) error
	IncreaseOccupiedCapacity(ctx context.Context, capacityID string) error
	DecreaseOccupiedCapacity(ctx context.Context, capacityID string) error
	UpdateDateCapacity(ctx context.Context, capacityID string, totalCapacity int64, roomWindowID string, roomWindowVersion int64) error
	GetBookingForUpdate(ctx context.Context, bookingID string) (Booking, error)
	CreateBooking(ctx context.Context, booking Booking) error
	CheckInBooking(ctx context.Context, booking Booking, expectedVersion int64) error
	DeleteBooking(ctx context.Context, bookingID string) error
}
