package input

import "hospital/service/appointment/rpc/internal/manager/common"

// CreateRoom 是创建房间时允许提交的严格地址字段。
type CreateRoom struct {
	DepartmentID string
	CampusID     string
	Building     string
	FloorNumber  int32
	RoomNumber   string
	Operation
}

// UpdateRoom 是修改房间严格地址时允许提交的字段。
type UpdateRoom struct {
	RoomID          string
	CampusID        string
	Building        string
	FloorNumber     int32
	RoomNumber      string
	ExpectedVersion int64
	Operation
}

// RetireRoom 是移除房间默认使用状态时允许提交的字段。
type RetireRoom struct {
	RoomID          string
	ExpectedVersion int64
	Operation
}

// ChangeStatus 是房间—项目关系或时间窗口状态变化的通用输入。
type ChangeStatus struct {
	ResourceID      string
	ExpectedVersion int64
	Operation
}

// AddRoomItem 是建立房间可执行项目关系时允许提交的字段。
type AddRoomItem struct {
	RoomID string
	ItemID string
	Operation
}

// SetRoomWindow 是设置房间开放时间和容量时允许提交的字段。
type SetRoomWindow struct {
	WindowID        string
	RoomID          string
	Weekday         int32
	Session         common.Session
	OpenTime        string
	CloseTime       string
	ActiveCapacity  int64
	ExpectedVersion int64
	Operation
}

// ListRooms 是按科室分页读取房间时允许提交的筛选字段。
type ListRooms struct {
	DepartmentID string
	Page         int64
	PageSize     int64
}

// ListRelations 是分页读取房间—项目关系时允许提交的筛选字段。
type ListRelations struct {
	Status   common.Status
	Page     int64
	PageSize int64
}
