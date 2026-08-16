package input

// PrepareProjectConfiguration 是 Guidance 在 TCC Try 阶段交给 Appointment 的完整项目事实。
type PrepareProjectConfiguration struct {
	TransactionID            string
	Action                   string
	ItemID                   string
	OwnerDepartmentID        string
	Name                     string
	Description              string
	EstimatedDurationMinutes int32
	ExpectedVersion          int64
	OperationID              string
	RequestID                string
}
