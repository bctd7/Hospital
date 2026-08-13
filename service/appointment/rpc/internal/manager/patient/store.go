package patient

import (
	"context"
	"time"

	"hospital/service/appointment/rpc/internal/manager/common"
)

// ProjectReader 只提供创建预约前确认项目状态所需的最小读取能力。
// 项目列表、详情、房间和窗口等双方共用能力统一由 shared.Manager 负责。
type ProjectReader interface {
	GetItemSummary(ctx context.Context, itemID string) (common.ItemSummary, error)
}

// BookingStore 包含患者查询、创建和取消本人预约所需的能力。
// 事务内部的细粒度锁和容量操作由共享 BookingTxStore 描述。
type BookingStore interface {
	ListBookingOptions(ctx context.Context, itemID string, fromDate, throughDate time.Time) ([]BookingOption, error)
	GetBooking(ctx context.Context, bookingID string) (Booking, error)
	ListBookings(ctx context.Context, filter BookingListFilter) ([]Booking, int64, error)
	WithinBookingTransaction(ctx context.Context, fn func(BookingTxStore) error) error
}
