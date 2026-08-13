// Package manager contains account, doctor, and self-profile business rules.
// Login and token issuance belong to authentication and session respectively.
package manager

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"hospital/common/authn"
)

const maxRequestIDBytes = 64

type Manager struct {
	store    Store
	phoneKey []byte
}

func NewManager(store Store, phoneKey []byte) (*Manager, error) {
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
	mutation func(TxStore, Account) error,
) (Account, []string, error) {
	if !operator.HasPermission(permission) {
		return Account{}, nil, ErrForbidden
	}
	accountID, err := normalizedUUID(accountID, "account_id")
	if err != nil {
		return Account{}, nil, err
	}
	operationID, err = normalizedUUID(operationID, "operation_id")
	if err != nil {
		return Account{}, nil, err
	}
	requestID = strings.TrimSpace(requestID)
	if len(requestID) > maxRequestIDBytes {
		return Account{}, nil, fmt.Errorf("%w: request_id exceeds %d bytes", ErrInvalid, maxRequestIDBytes)
	}
	if expectedVersion <= 0 {
		return Account{}, nil, fmt.Errorf("%w: management_version must be positive", ErrInvalid)
	}
	if operator.AccountID == accountID {
		return Account{}, nil, fmt.Errorf("%w: self management changes are not allowed", ErrForbidden)
	}

	var result Account
	err = m.store.WithinAccountTransaction(ctx, func(tx TxStore) error {
		operation, exists, err := tx.FindAccountOperation(ctx, operationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operator.AccountID || operation.TargetAccountID != accountID || operation.Action != action {
				return fmt.Errorf("%w: operation_id was already used for a different change", ErrConflict)
			}
			result, err = tx.GetAccountForUpdate(ctx, accountID)
			return err
		}

		before, err := tx.GetAccountForUpdate(ctx, accountID)
		if err != nil {
			return err
		}
		if before.IdentityType() == IdentityTypeSuperAdmin {
			return fmt.Errorf("%w: super administrator accounts are read-only", ErrForbidden)
		}
		if before.ManagementVersion != expectedVersion {
			return ErrVersionConflict
		}
		if err := mutation(tx, before); err != nil {
			return err
		}
		after, err := tx.GetAccountForUpdate(ctx, accountID)
		if err != nil {
			return err
		}
		if err := tx.RecordAccountChange(ctx, Change{
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
		return Account{}, nil, err
	}
	return result, availableActions(operator, result), nil
}
