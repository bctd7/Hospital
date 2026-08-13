package catalog

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusDisabled:
		return true
	default:
		return false
	}
}

// ExaminationItem is the current domain state of one examination catalog item.
// Transport requests and persistence-only records intentionally do not belong
// in this file.
type ExaminationItem struct {
	ItemID            string
	OwnerDepartmentID string
	Name              string
	Description       string
	Status            Status
	Version           int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
