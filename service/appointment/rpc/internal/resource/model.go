package resource

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

func (s Status) Valid() bool { return s == StatusActive || s == StatusDisabled }

type Session string

const (
	SessionMorning   Session = "morning"
	SessionAfternoon Session = "afternoon"
)

func (s Session) Valid() bool { return s == SessionMorning || s == SessionAfternoon }

type Room struct {
	RoomID       string    `json:"room_id"`
	DepartmentID string    `json:"department_id"`
	Name         string    `json:"name"`
	Status       Status    `json:"status"`
	Version      int64     `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ItemSummary struct {
	ItemID       string `json:"item_id"`
	DepartmentID string `json:"department_id"`
	Name         string `json:"name"`
	Status       Status `json:"status"`
	Version      int64  `json:"version"`
}

type RoomItem struct {
	RelationID string    `json:"relation_id"`
	RoomID     string    `json:"room_id"`
	ItemID     string    `json:"item_id"`
	RoomName   string    `json:"room_name,omitempty"`
	ItemName   string    `json:"item_name,omitempty"`
	Status     Status    `json:"status"`
	Version    int64     `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type RoomWeeklyWindow struct {
	WindowID       string    `json:"window_id"`
	RoomID         string    `json:"room_id"`
	Weekday        int32     `json:"weekday"`
	Session        Session   `json:"session"`
	OpenTime       string    `json:"open_time"`
	CloseTime      string    `json:"close_time"`
	ActiveCapacity int64     `json:"active_capacity"`
	Status         Status    `json:"status"`
	Version        int64     `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ItemWeeklyWindow struct {
	WindowID          string    `json:"window_id"`
	ItemID            string    `json:"item_id"`
	Weekday           int32     `json:"weekday"`
	Session           Session   `json:"session"`
	StartTime         string    `json:"start_time"`
	BookingCutoffTime string    `json:"booking_cutoff_time"`
	EndTime           string    `json:"end_time"`
	Status            Status    `json:"status"`
	Version           int64     `json:"version"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int64 `json:"page"`
	PageSize int64 `json:"page_size"`
	Total    int64 `json:"total"`
}
