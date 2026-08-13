package common

import "time"

type BookingStatus string

const (
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusCheckedIn BookingStatus = "checked_in"
)

func (s BookingStatus) Valid() bool {
	return s == BookingStatusConfirmed || s == BookingStatusCheckedIn
}

type Booking struct {
	BookingID         string        `json:"booking_id"`
	PatientAccountID  string        `json:"patient_account_id"`
	DepartmentID      string        `json:"department_id"`
	ItemID            string        `json:"item_id"`
	ItemName          string        `json:"item_name"`
	RoomID            string        `json:"room_id"`
	RoomDisplayName   string        `json:"room_display_name"`
	CampusID          string        `json:"campus_id"`
	ServiceDate       time.Time     `json:"service_date"`
	Session           Session       `json:"session"`
	Status            BookingStatus `json:"status"`
	RoomOpenTime      string        `json:"room_open_time"`
	RoomCloseTime     string        `json:"room_close_time"`
	ItemStartTime     string        `json:"item_start_time"`
	ItemEndTime       string        `json:"item_end_time"`
	BookingCutoffTime string        `json:"booking_cutoff_time"`
	CheckedInAt       *time.Time    `json:"checked_in_at,omitempty"`
	CheckedInBy       string        `json:"checked_in_by,omitempty"`
	Version           int64         `json:"version"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type BookingOption struct {
	ItemID            string    `json:"item_id"`
	DepartmentID      string    `json:"department_id"`
	RoomID            string    `json:"room_id"`
	RoomDisplayName   string    `json:"room_display_name"`
	CampusID          string    `json:"campus_id"`
	Building          string    `json:"building"`
	FloorNumber       int32     `json:"floor_number"`
	RoomNumber        string    `json:"room_number"`
	ServiceDate       time.Time `json:"service_date"`
	Session           Session   `json:"session"`
	RoomOpenTime      string    `json:"room_open_time"`
	RoomCloseTime     string    `json:"room_close_time"`
	ItemStartTime     string    `json:"item_start_time"`
	ItemEndTime       string    `json:"item_end_time"`
	BookingCutoffTime string    `json:"booking_cutoff_time"`
	TotalCapacity     int64     `json:"total_capacity"`
	OccupiedCapacity  int64     `json:"occupied_capacity"`
	RemainingCapacity int64     `json:"remaining_capacity"`
}

type BookingSelection struct {
	ItemID            string
	DepartmentID      string
	ItemName          string
	RoomID            string
	RoomDisplayName   string
	CampusID          string
	Session           Session
	RoomWindowID      string
	RoomWindowVersion int64
	RoomOpenTime      string
	RoomCloseTime     string
	ActiveCapacity    int64
	ItemStartTime     string
	ItemEndTime       string
	BookingCutoffTime string
}

type DateCapacity struct {
	CapacityID        string
	RoomID            string
	ServiceDate       time.Time
	Session           Session
	TotalCapacity     int64
	OccupiedCapacity  int64
	RoomWindowID      string
	RoomWindowVersion int64
	Version           int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type BookingListFilter struct {
	PatientAccountID string
	DepartmentID     string
	ServiceDate      *time.Time
	FromDate         *time.Time
	ThroughDate      *time.Time
	Session          Session
	ItemID           string
	RoomID           string
	Status           BookingStatus
	Offset           int64
	Limit            int64
}

type BookingOperation struct {
	OperatorAccountID  string
	BookingID          string
	Action             string
	RequestFingerprint string
	ResultData         []byte
}

type BookingOperationChange struct {
	OperationID        string
	OperatorAccountID  string
	BookingID          string
	Action             string
	RequestFingerprint string
	Result             any
}
