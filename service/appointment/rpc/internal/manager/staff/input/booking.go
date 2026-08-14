package input

import "hospital/service/appointment/rpc/internal/manager/common"

// ListBookings 是工作人员分页读取预约时允许提交的筛选字段。
type ListBookings struct {
	DepartmentID   string
	ServiceDate    string
	Session        common.Session
	ItemID         string
	RoomID         string
	Status         common.BookingStatus
	View           common.BookingListView
	PatientKeyword string
	Page           int64
	PageSize       int64
}

// DeleteBooking 是工作人员删除预约时允许提交的字段。
type DeleteBooking struct {
	BookingID   string
	OperationID string
	Reason      string
	RequestID   string
}
