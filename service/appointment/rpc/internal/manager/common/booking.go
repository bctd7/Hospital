package common

import "time"

type BookingStatus string

// BookingListView 是面向用户的预约生命周期视图，不改变底层预约状态机。
type BookingListView string

const (
	BookingStatusConfirmed  BookingStatus = "confirmed"
	BookingStatusInProgress BookingStatus = "in_progress"
	BookingStatusCompleted  BookingStatus = "completed"
	BookingStatusNoShow     BookingStatus = "no_show"
	BookingStatusCanceled   BookingStatus = "canceled"
)

const (
	BookingListViewActive    BookingListView = "active"
	BookingListViewCompleted BookingListView = "completed"
)

func (v BookingListView) Valid() bool {
	return v == BookingListViewActive || v == BookingListViewCompleted
}

func (s BookingStatus) Valid() bool {
	return s == BookingStatusConfirmed || s == BookingStatusInProgress ||
		s == BookingStatusCompleted || s == BookingStatusNoShow || s == BookingStatusCanceled
}

type Booking struct {
	BookingID              string        `json:"booking_id"`
	PatientAccountID       string        `json:"patient_account_id"`
	PatientDisplayName     string        `json:"patient_display_name"`
	PatientPhoneMasked     string        `json:"patient_phone_masked"`
	PatientPhoneLast4      string        `json:"-"`
	DepartmentID           string        `json:"department_id"`
	ItemID                 string        `json:"item_id"`
	ItemName               string        `json:"item_name"`
	RoomID                 string        `json:"room_id"`
	RoomDisplayName        string        `json:"room_display_name"`
	CampusID               string        `json:"campus_id"`
	Building               string        `json:"building"`
	FloorNumber            int32         `json:"floor_number"`
	RoomNumber             string        `json:"room_number"`
	ServiceDate            time.Time     `json:"service_date"`
	Session                Session       `json:"session"`
	Status                 BookingStatus `json:"status"`
	RoomOpenTime           string        `json:"room_open_time"`
	RoomCloseTime          string        `json:"room_close_time"`
	ItemStartTime          string        `json:"item_start_time"`
	ItemEndTime            string        `json:"item_end_time"`
	BookingCutoffTime      string        `json:"booking_cutoff_time"`
	StartedAt              *time.Time    `json:"started_at,omitempty"`
	StartedBy              string        `json:"started_by,omitempty"`
	StartedByDisplayName   string        `json:"started_by_display_name,omitempty"`
	CompletedAt            *time.Time    `json:"completed_at,omitempty"`
	CompletedBy            string        `json:"completed_by,omitempty"`
	CompletedByDisplayName string        `json:"completed_by_display_name,omitempty"`
	Version                int64         `json:"version"`
	CreatedAt              time.Time     `json:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at"`
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
	Building          string
	FloorNumber       int32
	RoomNumber        string
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
	PatientKeyword   string
	DepartmentID     string
	ServiceDate      *time.Time
	FromDate         *time.Time
	ThroughDate      *time.Time
	Session          Session
	ItemID           string
	RoomID           string
	Status           BookingStatus
	View             BookingListView
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
