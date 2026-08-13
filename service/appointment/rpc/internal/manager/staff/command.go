package staff

type OperationMeta struct {
	OperationID string
	RequestID   string
}

type CreateRoomCommand struct {
	DepartmentID string
	CampusID     string
	Building     string
	FloorNumber  int32
	RoomNumber   string
	OperationMeta
}

type UpdateRoomCommand struct {
	RoomID          string
	CampusID        string
	Building        string
	FloorNumber     int32
	RoomNumber      string
	ExpectedVersion int64
	OperationMeta
}

type RetireRoomCommand struct {
	RoomID          string
	ExpectedVersion int64
	OperationMeta
}

type ChangeStatusCommand struct {
	ResourceID      string
	ExpectedVersion int64
	OperationMeta
}

type AddRoomItemCommand struct {
	RoomID string
	ItemID string
	OperationMeta
}

type SetRoomWindowCommand struct {
	WindowID        string
	RoomID          string
	Weekday         int32
	Session         Session
	OpenTime        string
	CloseTime       string
	ActiveCapacity  int64
	ExpectedVersion int64
	OperationMeta
}

type SetItemWindowCommand struct {
	WindowID          string
	ItemID            string
	Weekday           int32
	Session           Session
	StartTime         string
	BookingCutoffTime string
	EndTime           string
	ExpectedVersion   int64
	OperationMeta
}

type ListRoomsQuery struct {
	DepartmentID string
	Page         int64
	PageSize     int64
}

type ListRelationsQuery struct {
	Status   Status
	Page     int64
	PageSize int64
}
