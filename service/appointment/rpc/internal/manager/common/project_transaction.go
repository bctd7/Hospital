package common

import "context"

type ProjectOperation struct {
	OperatorAccountID  string
	ItemID             string
	Action             string
	RequestFingerprint string
	Result             ExaminationItem
}

type ProjectChange struct {
	OperationID        string
	OperatorAccountID  string
	ItemID             string
	Action             string
	RequestFingerprint string
	Before             *ExaminationItem
	After              ExaminationItem
	RequestID          string
}

type ProjectListFilter struct {
	OwnerDepartmentID string
	Status            Status
	Offset            int64
	Limit             int64
}

// ProjectTxStore 是项目写事务内使用的原子操作集合。
// 业务入口使用的 Store 由 shared、staff 和 patient 根据各自用例定义，避免暴露无关能力。
type ProjectTxStore interface {
	BookingTxStore
	FindOperation(ctx context.Context, operationID string) (ProjectOperation, bool, error)
	GetItemForUpdate(ctx context.Context, itemID string) (ExaminationItem, error)
	CreateItem(ctx context.Context, item ExaminationItem) error
	UpdateItem(ctx context.Context, item ExaminationItem, expectedVersion int64) error
	UpdateItemReportTemplate(ctx context.Context, item ExaminationItem, expectedTemplateVersion int64) error
	SetItemStatus(ctx context.Context, item ExaminationItem, expectedVersion int64) error
	RecordChange(ctx context.Context, change ProjectChange) error
}
