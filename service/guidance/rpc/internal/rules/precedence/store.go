package precedence

import "context"

type ProjectDirectory interface {
	ResolveProject(ctx context.Context, itemID string) (ProjectReference, error)
}

type Store interface {
	ListAll(ctx context.Context) ([]Rule, error)
	WithinWriteTransaction(ctx context.Context, fn func(TxStore) error) error
}

type TxStore interface {
	LockGraph(ctx context.Context) error
	ListAll(ctx context.Context) ([]Rule, error)
	GetByIDForUpdate(ctx context.Context, ruleID string) (Rule, error)
	GetByPair(ctx context.Context, predecessorItemID, successorItemID string) (Rule, error)
	GetByCreateOperationID(ctx context.Context, operationID string) (Rule, error)
	Insert(ctx context.Context, rule Rule) (Rule, error)
	UpdateText(ctx context.Context, ruleID, staffReason, patientMessage string, expectedVersion int64) (Rule, error)
	Delete(ctx context.Context, ruleID string, expectedVersion int64) error
}
