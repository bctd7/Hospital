package common

import (
	"context"
	"time"
)

// QueueTxStore 在预约事务能力之上，仅补充检查报到和房间叫号需要的原子操作。
// 它与项目配置事务分离，避免项目管理被迫依赖队列实现。
type QueueTxStore interface {
	BookingTxStore
	CheckInBooking(ctx context.Context, booking Booking, expectedVersion int64, queueID, eventID string) (QueueEntry, error)
	ListRoomQueuesForUpdate(ctx context.Context, roomID string, serviceDate time.Time) ([]CheckQueue, error)
	HasInProgressBookingForUpdate(ctx context.Context, roomID string, serviceDate time.Time) (bool, error)
	FindNextQueueBookingForUpdate(ctx context.Context, queue CheckQueue, now time.Time) (Booking, QueueEntry, bool, error)
	GetQueueEntryForUpdate(ctx context.Context, bookingID string) (QueueEntry, error)
	MarkBookingCalled(ctx context.Context, booking Booking, queue CheckQueue, entry QueueEntry, expectedVersion int64, eventID string) error
	MarkBookingDeferred(ctx context.Context, booking Booking, queue CheckQueue, entry QueueEntry, eventID string) error
	RecordQueueEvent(ctx context.Context, event QueueEvent) error
}
