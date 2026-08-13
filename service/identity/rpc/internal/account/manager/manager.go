// Package manager 编排账号、医生和本人资料的业务操作。
// 领域模型、稳定错误和 Store 端口属于父包 account；登录与 Token 分别属于 authentication 和 session。
package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/account"
)

const maxRequestIDBytes = 64

type Manager struct {
	store    account.Store
	phoneKey []byte
}

func NewManager(store account.Store, phoneKey []byte) (*Manager, error) {
	if store == nil || len(phoneKey) < 32 {
		return nil, errors.New("identity admin manager dependencies are required")
	}
	return &Manager{store: store, phoneKey: append([]byte(nil), phoneKey...)}, nil
}

func (m *Manager) mutate(
	ctx context.Context,
	operator authn.Principal,
	accountID string,
	expectedVersion int64,
	operationID, requestID, action, permission string,
	mutation func(account.TxStore, account.Account) error,
) (account.Account, []string, error) {
	if !operator.HasPermission(permission) {
		return account.Account{}, nil, account.ErrForbidden
	}
	accountID, err := normalizedUUID(accountID, "account_id")
	if err != nil {
		return account.Account{}, nil, err
	}
	operationID, err = normalizedUUID(operationID, "operation_id")
	if err != nil {
		return account.Account{}, nil, err
	}
	requestID = strings.TrimSpace(requestID)
	if len(requestID) > maxRequestIDBytes {
		return account.Account{}, nil, fmt.Errorf("%w: request_id exceeds %d bytes", account.ErrInvalid, maxRequestIDBytes)
	}
	if expectedVersion <= 0 {
		return account.Account{}, nil, fmt.Errorf("%w: management_version must be positive", account.ErrInvalid)
	}
	if operator.AccountID == accountID {
		return account.Account{}, nil, fmt.Errorf("%w: self management changes are not allowed", account.ErrForbidden)
	}

	var result account.Account
	err = m.store.WithinAccountTransaction(ctx, func(tx account.TxStore) error {
		operation, exists, err := tx.FindAccountOperation(ctx, operationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operator.AccountID || operation.TargetAccountID != accountID || operation.Action != action {
				return fmt.Errorf("%w: operation_id was already used for a different change", account.ErrConflict)
			}
			result, err = tx.GetAccountForUpdate(ctx, accountID)
			return err
		}

		before, err := tx.GetAccountForUpdate(ctx, accountID)
		if err != nil {
			return err
		}
		if before.IdentityType() == account.IdentityTypeSuperAdmin {
			return fmt.Errorf("%w: super administrator accounts are read-only", account.ErrForbidden)
		}
		if before.ManagementVersion != expectedVersion {
			return account.ErrVersionConflict
		}
		if err := mutation(tx, before); err != nil {
			return err
		}
		after, err := tx.GetAccountForUpdate(ctx, accountID)
		if err != nil {
			return err
		}
		if err := tx.RecordAccountChange(ctx, account.Change{
			OperationID: operationID, OperatorAccountID: operator.AccountID,
			TargetAccountID: accountID, Action: action, Before: before, After: after,
			RequestID: requestID,
		}); err != nil {
			return err
		}
		result = after
		return nil
	})
	if err != nil {
		return account.Account{}, nil, err
	}
	return result, availableActions(operator, result), nil
}
