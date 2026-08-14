package common

import "time"

// ReportStatus 表示一次检查报告主体当前是否已经存在正式版本。
type ReportStatus string

const (
	ReportStatusDraft     ReportStatus = "draft"
	ReportStatusPublished ReportStatus = "published"
)

// ReportVersionStatus 区分可编辑草稿、当前正式版和已被更正替代的历史正式版。
type ReportVersionStatus string

const (
	ReportVersionStatusDraft      ReportVersionStatus = "draft"
	ReportVersionStatusPublished  ReportVersionStatus = "published"
	ReportVersionStatusSuperseded ReportVersionStatus = "superseded"
)

type ReportVersionKind string

const (
	ReportVersionKindInitial    ReportVersionKind = "initial"
	ReportVersionKindCorrection ReportVersionKind = "correction"
)

// ReportContent 是医生实际填写的固定报告表单；身份、项目和地址由系统快照生成。
type ReportContent struct {
	ObjectiveFindings string `json:"objective_findings"`
	Impression        string `json:"impression"`
	Recommendation    string `json:"recommendation"`
	Notes             string `json:"notes"`
}

type ReportVersion struct {
	VersionID   string              `json:"version_id"`
	ReportID    string              `json:"report_id"`
	VersionNo   int64               `json:"version_no"`
	VersionKind ReportVersionKind   `json:"version_kind"`
	Status      ReportVersionStatus `json:"status"`
	ReportContent
	CorrectionReason       string     `json:"correction_reason,omitempty"`
	AuthoredBy             string     `json:"authored_by"`
	AuthoredByDisplayName  string     `json:"authored_by_display_name"`
	PublishedBy            string     `json:"published_by,omitempty"`
	PublishedByDisplayName string     `json:"published_by_display_name,omitempty"`
	PublishedAt            *time.Time `json:"published_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// ExaminationReport 是一次预约对应的报告主体。CurrentVersion 对患者始终是当前正式版。
type ExaminationReport struct {
	ReportID               string         `json:"report_id"`
	BookingID              string         `json:"booking_id"`
	PatientAccountID       string         `json:"patient_account_id"`
	PatientDisplayName     string         `json:"patient_display_name"`
	PatientPhoneMasked     string         `json:"patient_phone_masked"`
	DepartmentID           string         `json:"department_id"`
	DepartmentName         string         `json:"department_name"`
	ItemID                 string         `json:"item_id"`
	ItemName               string         `json:"item_name"`
	RoomID                 string         `json:"room_id"`
	CampusID               string         `json:"campus_id"`
	CampusName             string         `json:"campus_name"`
	Building               string         `json:"building"`
	FloorNumber            int32          `json:"floor_number"`
	RoomNumber             string         `json:"room_number"`
	RoomDisplayName        string         `json:"room_display_name"`
	Status                 ReportStatus   `json:"status"`
	PerformedBy            string         `json:"performed_by,omitempty"`
	PerformedByDisplayName string         `json:"performed_by_display_name,omitempty"`
	ExaminationStartedAt   *time.Time     `json:"examination_started_at,omitempty"`
	ExaminationCompletedAt *time.Time     `json:"examination_completed_at,omitempty"`
	CurrentVersionID       string         `json:"current_version_id,omitempty"`
	CurrentVersion         *ReportVersion `json:"current_version,omitempty"`
	Version                int64          `json:"version"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}

type ReportListFilter struct {
	PatientAccountID string
	DepartmentID     string
	Keyword          string
	Status           ReportStatus
	Offset           int64
	Limit            int64
}
