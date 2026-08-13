package catalog

// Commands describe manager use-case input. They are separate from the domain
// model because callers do not control generated IDs, status, versions, or
// timestamps.
type CreateCommand struct {
	OwnerDepartmentID string
	Name              string
	Description       string
	OperationID       string
	RequestID         string
}

type UpdateCommand struct {
	ItemID          string
	Name            *string
	Description     *string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type ChangeStatusCommand struct {
	ItemID          string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type ListQuery struct {
	OwnerDepartmentID string
	Status            Status
	Page              int64
	PageSize          int64
	RequestID         string
}

type ListResult struct {
	Items    []ExaminationItem
	Page     int64
	PageSize int64
	Total    int64
}
