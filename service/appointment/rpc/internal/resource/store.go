package resource

import "context"

type Operation struct {
	OperatorAccountID  string
	ResourceType       string
	ResourceID         string
	Action             string
	RequestFingerprint string
	ResultData         []byte
}

type Change struct {
	OperationID        string
	OperatorAccountID  string
	DepartmentID       string
	ResourceType       string
	ResourceID         string
	Action             string
	RequestFingerprint string
	Before             any
	After              any
	RequestID          string
}

type Store interface {
	GetRoom(ctx context.Context, roomID string) (Room, error)
	ListRooms(ctx context.Context, departmentID string, status Status, offset, limit int64) ([]Room, int64, error)
	GetItemSummary(ctx context.Context, itemID string) (ItemSummary, error)
	ListRoomItems(ctx context.Context, roomID string, status Status, offset, limit int64) ([]RoomItem, int64, error)
	ListItemRooms(ctx context.Context, itemID string, activeOnly bool) ([]RoomItem, error)
	ListRoomWindows(ctx context.Context, roomID string, activeOnly bool) ([]RoomWeeklyWindow, error)
	ListItemWindows(ctx context.Context, itemID string, activeOnly bool) ([]ItemWeeklyWindow, error)
	WithinResourceTransaction(ctx context.Context, fn func(TxStore) error) error
}

type TxStore interface {
	FindOperation(ctx context.Context, operationID string) (Operation, bool, error)
	RecordChange(ctx context.Context, change Change) error

	GetRoomForUpdate(ctx context.Context, roomID string) (Room, error)
	CreateRoom(ctx context.Context, room Room) error
	UpdateRoom(ctx context.Context, room Room, expectedVersion int64) error
	SetRoomStatus(ctx context.Context, room Room, expectedVersion int64) error
	GetItemForUpdate(ctx context.Context, itemID string) (ItemSummary, error)

	GetRelationForUpdate(ctx context.Context, relationID string) (RoomItem, error)
	FindRelationForUpdate(ctx context.Context, roomID, itemID string) (RoomItem, bool, error)
	CreateRelation(ctx context.Context, relation RoomItem) error
	SetRelationStatus(ctx context.Context, relation RoomItem, expectedVersion int64) error
	ListRoomRelationsForUpdate(ctx context.Context, roomID string) ([]RoomItem, error)
	ListItemRelationsForUpdate(ctx context.Context, itemID string) ([]RoomItem, error)

	GetRoomWindowForUpdate(ctx context.Context, windowID string) (RoomWeeklyWindow, error)
	FindRoomWindowForUpdate(ctx context.Context, roomID string, weekday int32, session Session) (RoomWeeklyWindow, bool, error)
	CreateRoomWindow(ctx context.Context, window RoomWeeklyWindow) error
	UpdateRoomWindow(ctx context.Context, window RoomWeeklyWindow, expectedVersion int64) error
	SetRoomWindowStatus(ctx context.Context, window RoomWeeklyWindow, expectedVersion int64) error
	ListRoomWindowsForUpdate(ctx context.Context, roomID string) ([]RoomWeeklyWindow, error)

	GetItemWindowForUpdate(ctx context.Context, windowID string) (ItemWeeklyWindow, error)
	FindItemWindowForUpdate(ctx context.Context, itemID string, weekday int32, session Session) (ItemWeeklyWindow, bool, error)
	CreateItemWindow(ctx context.Context, window ItemWeeklyWindow) error
	UpdateItemWindow(ctx context.Context, window ItemWeeklyWindow, expectedVersion int64) error
	SetItemWindowStatus(ctx context.Context, window ItemWeeklyWindow, expectedVersion int64) error
	ListItemWindowsForUpdate(ctx context.Context, itemID string) ([]ItemWeeklyWindow, error)
}
