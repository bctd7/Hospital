package identityadmin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"hospital/common/authn"
)

const maxRequestIDBytes = 64

type AuthorizationVersionPublisher interface {
	SetAuthorizationVersion(ctx context.Context, accountID string, version int64) error
}

type Manager struct {
	store    Store
	versions AuthorizationVersionPublisher
	phoneKey []byte
}

func NewManager(store Store, versions AuthorizationVersionPublisher, phoneKey []byte) (*Manager, error) {
	if store == nil || versions == nil || len(phoneKey) < 32 {
		return nil, errors.New("identity admin manager dependencies are required")
	}
	return &Manager{store: store, versions: versions, phoneKey: append([]byte(nil), phoneKey...)}, nil
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
	var beforeVersion int64
	var replayed bool
	err = m.store.WithinIdentityAdminTransaction(ctx, func(tx TxStore) error {
		operation, exists, err := tx.FindIdentityAdminOperation(ctx, operationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operator.AccountID || operation.TargetAccountID != accountID || operation.Action != action {
				return fmt.Errorf("%w: operation_id was already used for a different change", ErrConflict)
			}
			result, err = tx.GetIdentityAdminAccountForUpdate(ctx, accountID)
			replayed = true
			return err
		}

		before, err := tx.GetIdentityAdminAccountForUpdate(ctx, accountID)
		if err != nil {
			return err
		}
		if before.IdentityType() == IdentityTypeSuperAdmin {
			return fmt.Errorf("%w: super administrator accounts are read-only", ErrForbidden)
		}
		if before.ManagementVersion != expectedVersion {
			return ErrVersionConflict
		}
		beforeVersion = before.AuthorizationVersion
		if err := mutation(tx, before); err != nil {
			return err
		}
		after, err := tx.GetIdentityAdminAccountForUpdate(ctx, accountID)
		if err != nil {
			return err
		}
		if err := tx.RecordIdentityAdminChange(ctx, Change{
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
	if (!replayed && result.AuthorizationVersion != beforeVersion) ||
		(replayed && actionChangesAuthorization(action)) {
		if err := m.versions.SetAuthorizationVersion(ctx, result.ID, result.AuthorizationVersion); err != nil {
			return Account{}, nil, fmt.Errorf("publish authorization version: %w", err)
		}
	}
	return result, availableActions(operator, result), nil
}

func actionChangesAuthorization(action string) bool {
	switch action {
	case ActionPromoteDoctor, ActionChangeDoctorDepartment, ActionRevokeDoctor,
		ActionDisableAccount, ActionEnableAccount:
		return true
	default:
		return false
	}
}
