package common

import "context"

type ConfigurationOperation struct {
	OperatorAccountID  string
	ResourceType       string
	ResourceID         string
	Action             string
	RequestFingerprint string
	ResultData         []byte
}

type ConfigurationChange struct {
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

// ConfigurationTxStore 是房间、项目关系和周配置写事务内的原子操作集合。
type ConfigurationTxStore interface {
	BookingTxStore
	FindOperation(ctx context.Context, operationID string) (ConfigurationOperation, bool, error)
	RecordChange(ctx context.Context, change ConfigurationChange) error
	GetRoomForUpdate(ctx context.Context, roomID string) (Room, error)
	CreateRoom(ctx context.Context, room Room) error
	UpdateRoom(ctx context.Context, room Room, expectedVersion int64) error
	RetireRoom(ctx context.Context, room Room, expectedVersion int64) error
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
