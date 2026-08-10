package organization

import "context"

// Operation describes an organization operation that has already completed.
// It is used to make retries with the same operation ID idempotent.
type Operation struct {
	OperatorAccountID string
	UnitID            string
	Action            string
	Result            Unit
}

// Change contains the stable facts needed for audit and Outbox records.
// Before is nil when a new organization unit is created.
type Change struct {
	OperationID       string
	OperatorAccountID string
	UnitID            string
	Action            string
	Before            *Unit
	After             Unit
	RequestID         string
}

// Store exposes ordinary organization reads and the organization transaction
// boundary. The authenticated operator is supplied to Manager as a Principal,
// so this port does not query operator roles or permissions again.
type Store interface {
	GetUnit(ctx context.Context, unitID string) (Unit, error)
	ListUnits(ctx context.Context, filter ListFilter) ([]Unit, error)
	WithinOrganizationTransaction(ctx context.Context, fn func(TxStore) error) error
}

// TxStore exposes only operations that must participate in the same database
// transaction as an organization change.
type TxStore interface {
	FindOperation(ctx context.Context, operationID string) (Operation, bool, error)
	GetUnitForUpdate(ctx context.Context, unitID string) (Unit, error)
	CreateUnit(ctx context.Context, unit Unit) error
	UpdateUnit(ctx context.Context, unit Unit, expectedVersion int64) error
	SetUnitStatus(ctx context.Context, unitID string, status Status, expectedVersion int64) error
	CountActiveChildren(ctx context.Context, unitID string) (int64, error)
	CountActiveDoctors(ctx context.Context, departmentID string) (int64, error)

	// RecordChange writes the audit record and its Outbox event atomically with
	// the organization mutation. Their SQL details remain in Repository.
	RecordChange(ctx context.Context, change Change) error
}
