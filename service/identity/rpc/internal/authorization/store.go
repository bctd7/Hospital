package authorization

import (
	"context"
	"errors"

	"hospital/common/authn"
)

var (
	ErrNotFound  = errors.New("identity resource not found")
	ErrConflict  = errors.New("identity operation conflicts with existing data")
	ErrInvalid   = errors.New("invalid identity authorization request")
	ErrForbidden = errors.New("identity authorization operation is forbidden")
)

type Change struct {
	OperationID       string
	OperatorAccountID string
	TargetAccountID   string
	Action            string
	Before            authn.Principal
	After             authn.Principal
	RequestID         string
}

type Operation struct {
	OperatorAccountID string
	TargetAccountID   string
}

type Store interface {
	GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error)
	WithinTransaction(ctx context.Context, fn func(TxStore) error) error
}

type TxStore interface {
	GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error)
	FindOperation(ctx context.Context, operationID string) (Operation, bool, error)
	SetRole(ctx context.Context, accountID, roleCode string) error
	SetDepartment(ctx context.Context, accountID, departmentID string) error
	SetAccountStatus(ctx context.Context, accountID, status string) error
	RecordChange(ctx context.Context, change Change) error
}
