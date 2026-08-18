// Package planning 生成患者智能预约宏观方案和当日检查顺序建议。
package planning

import "time"

type Option struct {
	ItemID, RoomID, RoomDisplayName, CampusID, Building, RoomNumber string
	FloorNumber, EstimatedDurationMinutes                           int32
	ServiceDate, Session                                            string
	RoomOpenTime, RoomCloseTime, ItemStartTime, ItemEndTime         string
	RemainingCapacity                                               int64
}

type Booking struct {
	BookingID, ItemID, ItemName, RoomID, RoomDisplayName, CampusID, Building, RoomNumber string
	FloorNumber, EstimatedDurationMinutes                                                int32
	ServiceDate, Session, Status                                                         string
}

type PlanItem struct {
	ItemID                   string `json:"item_id"`
	ItemName                 string `json:"item_name"`
	RoomID                   string `json:"room_id"`
	RoomDisplayName          string `json:"room_display_name"`
	CampusID                 string `json:"campus_id"`
	Building                 string `json:"building"`
	RoomNumber               string `json:"room_number"`
	FloorNumber              int32  `json:"floor_number"`
	EstimatedDurationMinutes int32  `json:"estimated_duration_minutes"`
	ServiceDate              string `json:"service_date"`
	Session                  string `json:"session"`
	PlannedStartTime         string `json:"planned_start_time,omitempty"`
	PlannedEndTime           string `json:"planned_end_time,omitempty"`
	TravelMinutes            int32  `json:"travel_minutes,omitempty"`
	TravelTimeEstimated      bool   `json:"travel_time_estimated,omitempty"`
	Reason                   string `json:"reason"`
}

type Plan struct {
	PlanID, PatientAccountID, Title, Summary string
	Items                                    []PlanItem
	BookingIDs                               []string
	ExpiresAt                                time.Time
	ConfirmedAt                              *time.Time
}

type Stage struct {
	StageNo              int32
	Title, Status, Focus string
	Items                []PlanItem
}

type TodayRecommendation struct {
	ServiceDate string
	Stages      []Stage
	UpdatedAt   time.Time
}

type GenerateCommand struct {
	ItemIDs               []string
	CandidateDates        []string
	CandidateAvailability []CandidateAvailability
}

type CandidateAvailability struct {
	ServiceDate string
	Sessions    []string
}

type ConfirmCommand struct {
	PlanID, OperationID, RequestID, PatientDisplayName, PatientPhoneMasked string
}
