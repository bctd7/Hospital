package common

// Status 表示项目、关系和周配置是否参与当前业务。
type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

func (s Status) Valid() bool { return s == StatusActive || s == StatusDisabled }

// Session 表示一天内固定的上午或下午预约段。
type Session string

const (
	SessionMorning   Session = "morning"
	SessionAfternoon Session = "afternoon"
)

func (s Session) Valid() bool { return s == SessionMorning || s == SessionAfternoon }

// Page 是各业务入口共用的分页结果。
type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int64 `json:"page"`
	PageSize int64 `json:"page_size"`
	Total    int64 `json:"total"`
}
