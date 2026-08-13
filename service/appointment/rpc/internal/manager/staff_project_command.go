package manager

// Commands describe manager use-case input. They are separate from the domain
// model because callers do not control generated IDs, status, versions, or
// timestamps.
type CreateProjectCommand struct {
	OwnerDepartmentID string
	Name              string
	Description       string
	OperationID       string
	RequestID         string
}

type UpdateProjectCommand struct {
	ItemID          string
	Name            *string
	Description     *string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type ChangeProjectStatusCommand struct {
	ItemID          string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type ListProjectsQuery struct {
	OwnerDepartmentID string
	Status            Status
	Page              int64
	PageSize          int64
	RequestID         string
}

type ListProjectsResult struct {
	Items    []ExaminationItem
	Page     int64
	PageSize int64
	Total    int64
}
