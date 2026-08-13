package manager

import "context"

type ProjectOperation struct {
	OperatorAccountID  string
	ItemID             string
	Action             string
	RequestFingerprint string
	Result             ExaminationItem
}

type ProjectChange struct {
	OperationID        string
	OperatorAccountID  string
	ItemID             string
	Action             string
	RequestFingerprint string
	Before             *ExaminationItem
	After              ExaminationItem
	RequestID          string
}

type ProjectListFilter struct {
	OwnerDepartmentID string
	Status            Status
	Offset            int64
	Limit             int64
}

type ProjectStore interface {
	GetItem(ctx context.Context, itemID string) (ExaminationItem, error)
	ListItems(ctx context.Context, filter ProjectListFilter) ([]ExaminationItem, int64, error)
	WithinProjectTransaction(ctx context.Context, fn func(ProjectTxStore) error) error
}

type ProjectTxStore interface {
	FindOperation(ctx context.Context, operationID string) (ProjectOperation, bool, error)
	GetItemForUpdate(ctx context.Context, itemID string) (ExaminationItem, error)
	CreateItem(ctx context.Context, item ExaminationItem) error
	UpdateItem(ctx context.Context, item ExaminationItem, expectedVersion int64) error
	SetItemStatus(ctx context.Context, item ExaminationItem, expectedVersion int64) error
	RecordChange(ctx context.Context, change ProjectChange) error
}

type RoomScheduleOperation struct {
	OperatorAccountID  string
	ResourceType       string
	ResourceID         string
	Action             string
	RequestFingerprint string
	ResultData         []byte
}

type RoomScheduleChange struct {
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

type RoomScheduleStore interface {
	GetRoom(ctx context.Context, roomID string) (Room, error)
	ListRooms(ctx context.Context, departmentID string, offset, limit int64) ([]Room, int64, error)
	GetItemSummary(ctx context.Context, itemID string) (ItemSummary, error)
	ListRoomItems(ctx context.Context, roomID string, status Status, offset, limit int64) ([]RoomItem, int64, error)
	ListItemRooms(ctx context.Context, itemID string, activeOnly bool) ([]RoomItem, error)
	ListRoomWindows(ctx context.Context, roomID string, activeOnly bool) ([]RoomWeeklyWindow, error)
	ListItemWindows(ctx context.Context, itemID string, activeOnly bool) ([]ItemWeeklyWindow, error)
	WithinRoomScheduleTransaction(ctx context.Context, fn func(RoomScheduleTxStore) error) error
}

type RoomScheduleTxStore interface {
	FindOperation(ctx context.Context, operationID string) (RoomScheduleOperation, bool, error)
	RecordChange(ctx context.Context, change RoomScheduleChange) error
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
