package common

import (
	"context"
	"time"
)

// BookingTxStore 是预约创建、开始检查、删除和容量变更共享的事务原子操作。
// 患者端与工作人员端分别在自己的 store.go 中声明所需的对外查询能力。
type BookingTxStore interface {
	FindBookingOperation(ctx context.Context, operationID string) (BookingOperation, bool, error)
	RecordBookingOperation(ctx context.Context, change BookingOperationChange) error
	ListBookingsForUpdate(ctx context.Context, filter BookingListFilter) ([]Booking, error)
	ClaimPatientSession(ctx context.Context, patientAccountID string, serviceDate time.Time, session Session, bookingID string) error
	ReleasePatientSession(ctx context.Context, bookingID string) error
	ConsumePatientWeeklyQuota(ctx context.Context, patientAccountID string, weekStartDate time.Time, limit int64, now time.Time) error
	LockBookingSelection(ctx context.Context, itemID, roomID string, serviceDate time.Time, session Session) (BookingSelection, error)
	FindDateCapacityForUpdate(ctx context.Context, roomID string, serviceDate time.Time, session Session) (DateCapacity, bool, error)
	CreateDateCapacity(ctx context.Context, capacity DateCapacity) error
	IncreaseOccupiedCapacity(ctx context.Context, capacityID string) error
	DecreaseOccupiedCapacity(ctx context.Context, capacityID string) error
	UpdateDateCapacity(ctx context.Context, capacityID string, totalCapacity int64, roomWindowID string, roomWindowVersion int64) error
	GetBookingForUpdate(ctx context.Context, bookingID string) (Booking, error)
	CreateBooking(ctx context.Context, booking Booking) error
	MarkBookingNoShow(ctx context.Context, booking Booking, expectedVersion int64) error
	DeleteBooking(ctx context.Context, bookingID string) error
}
