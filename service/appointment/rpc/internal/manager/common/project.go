package common

import "time"

// ExaminationItem 是检查项目的完整业务数据。
type ExaminationItem struct {
	ItemID                   string
	OwnerDepartmentID        string
	Name                     string
	Description              string
	EstimatedDurationMinutes int32
	ReportTemplate           ReportTemplate
	Status                   Status
	Version                  int64
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// ReportTemplate 是检查项目为新报告提供的固定四字段初始正文；版本 0 表示尚未配置。
type ReportTemplate struct {
	ReportContent
	Version int64 `json:"version"`
}

// ItemSummary 是校验项目归属和状态时使用的最小数据。
type ItemSummary struct {
	ItemID                   string `json:"item_id"`
	DepartmentID             string `json:"department_id"`
	Name                     string `json:"name"`
	EstimatedDurationMinutes int32  `json:"estimated_duration_minutes"`
	Status                   Status `json:"status"`
	Version                  int64  `json:"version"`
}

// ItemWeeklyWindow 表示检查项目自身的预约时间，必须完整落在房间开放窗口内。
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
