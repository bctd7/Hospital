package staff

import "context"

// ProjectWriteStore 是项目基本信息写事务需要的最小入口。
type ProjectWriteStore interface {
	WithinProjectTransaction(ctx context.Context, fn func(ProjectTxStore) error) error
}

// ProjectStore 是工作人员维护检查项目及其预约时间所需的完整入口。
// 面向页面的普通项目读取仍统一由 shared.Manager 处理。
type ProjectStore interface {
	ProjectWriteStore
	GetItemSummary(ctx context.Context, itemID string) (ItemSummary, error)
	ListItemRooms(ctx context.Context, itemID string, activeOnly bool) ([]RoomItem, error)
	ListItemWindows(ctx context.Context, itemID string, activeOnly bool) ([]ItemWeeklyWindow, error)
	WithinConfigurationTransaction(ctx context.Context, fn func(ConfigurationTxStore) error) error
}

// RoomStore 是工作人员维护房间、可执行项目关系和房间开放时间所需的入口。
type RoomStore interface {
	GetRoom(ctx context.Context, roomID string) (Room, error)
	ListRooms(ctx context.Context, departmentID string, offset, limit int64) ([]Room, int64, error)
	ListRoomItems(ctx context.Context, roomID string, status Status, offset, limit int64) ([]RoomItem, int64, error)
	ListRoomWindows(ctx context.Context, roomID string, activeOnly bool) ([]RoomWeeklyWindow, error)
	WithinConfigurationTransaction(ctx context.Context, fn func(ConfigurationTxStore) error) error
}

// BookingStore 是工作人员查询、核销和删除预约所需的入口。
type BookingStore interface {
	GetBooking(ctx context.Context, bookingID string) (Booking, error)
	ListBookings(ctx context.Context, filter BookingListFilter) ([]Booking, int64, error)
	WithinBookingTransaction(ctx context.Context, fn func(BookingTxStore) error) error
}
