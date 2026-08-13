package catalog

import "context"

// Operation is the stored result of a completed idempotent command.
type Operation struct {
	OperatorAccountID  string
	ItemID             string
	Action             string
	RequestFingerprint string
	Result             ExaminationItem
}

// Change contains the before/after facts written to the audit record in the
// same transaction as the catalog mutation.
type Change struct {
	OperationID        string
	OperatorAccountID  string
	ItemID             string
	Action             string
	RequestFingerprint string
	Before             *ExaminationItem
	After              ExaminationItem
	RequestID          string
}

// ListFilter is a persistence query, unlike ListQuery which is the manager's
// page-based use-case input.
type ListFilter struct {
	OwnerDepartmentID string
	Status            Status
	Offset            int64
	Limit             int64
}

type Store interface {
	GetItem(ctx context.Context, itemID string) (ExaminationItem, error)
	ListItems(ctx context.Context, filter ListFilter) ([]ExaminationItem, int64, error)
	WithinCatalogTransaction(ctx context.Context, fn func(TxStore) error) error
}

// TxStore contains only operations that must be atomic with idempotency and
// audit records. Its MySQL implementation is added with the catalog migration.
type TxStore interface {
	FindOperation(ctx context.Context, operationID string) (Operation, bool, error)
	GetItemForUpdate(ctx context.Context, itemID string) (ExaminationItem, error)
	CreateItem(ctx context.Context, item ExaminationItem) error
	UpdateItem(ctx context.Context, item ExaminationItem, expectedVersion int64) error
	SetItemStatus(ctx context.Context, item ExaminationItem, expectedVersion int64) error
	RecordChange(ctx context.Context, change Change) error
}
