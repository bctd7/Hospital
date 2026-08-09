package authorization

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
)

const (
	ActionRoleAssigned         = "identity.role.assigned"
	ActionDepartmentChanged    = "identity.department.changed"
	ActionAccountStatusChanged = "identity.account.status_changed"
)

type Manager struct {
	store Store
}

func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

func (m *Manager) GetAuthorizationContext(ctx context.Context, operatorID, accountID string) (authn.Principal, error) {
	if err := validateID(operatorID, "operator_account_id"); err != nil {
		return authn.Principal{}, err
	}
	if err := validateID(accountID, "account_id"); err != nil {
		return authn.Principal{}, err
	}
	if operatorID != accountID {
		operator, err := m.store.GetAuthorizationContext(ctx, operatorID)
		if err != nil {
			return authn.Principal{}, err
		}
		if err := commonauthz.RequirePermission(operator, contractauthz.PermissionIdentityAuthorizationManage); err != nil {
			return authn.Principal{}, fmt.Errorf("%w: %v", ErrForbidden, err)
		}
	}
	return m.store.GetAuthorizationContext(ctx, accountID)
}

func (m *Manager) AssignRole(ctx context.Context, operatorID, targetID, roleCode, operationID, requestID string) (authn.Principal, error) {
	if roleCode != authn.RoleSuperAdmin && roleCode != authn.RoleDepartmentDoctor {
		return authn.Principal{}, fmt.Errorf("%w: unsupported role %q", ErrInvalid, roleCode)
	}
	return m.change(ctx, operatorID, targetID, operationID, requestID, ActionRoleAssigned, func(tx TxStore, target authn.Principal) error {
		if target.AccountType != authn.AccountTypeStaff {
			return fmt.Errorf("%w: roles can only be assigned to staff accounts", ErrInvalid)
		}
		return tx.SetRole(ctx, targetID, roleCode)
	})
}

func (m *Manager) ChangeStaffDepartment(ctx context.Context, operatorID, targetID, departmentID, operationID, requestID string) (authn.Principal, error) {
	if err := validateID(departmentID, "department_id"); err != nil {
		return authn.Principal{}, err
	}
	return m.change(ctx, operatorID, targetID, operationID, requestID, ActionDepartmentChanged, func(tx TxStore, target authn.Principal) error {
		if target.AccountType != authn.AccountTypeStaff {
			return fmt.Errorf("%w: departments can only be assigned to staff accounts", ErrInvalid)
		}
		return tx.SetDepartment(ctx, targetID, departmentID)
	})
}

func (m *Manager) ChangeAccountStatus(ctx context.Context, operatorID, targetID, status, operationID, requestID string) (authn.Principal, error) {
	if status != authn.AccountStatusActive && status != authn.AccountStatusDisabled {
		return authn.Principal{}, fmt.Errorf("%w: unsupported account status %q", ErrInvalid, status)
	}
	return m.change(ctx, operatorID, targetID, operationID, requestID, ActionAccountStatusChanged, func(tx TxStore, _ authn.Principal) error {
		return tx.SetAccountStatus(ctx, targetID, status)
	})
}

func (m *Manager) change(
	ctx context.Context,
	operatorID, targetID, operationID, requestID, action string,
	mutate func(TxStore, authn.Principal) error,
) (authn.Principal, error) {
	if err := validateID(operatorID, "operator_account_id"); err != nil {
		return authn.Principal{}, err
	}
	if err := validateID(targetID, "target_account_id"); err != nil {
		return authn.Principal{}, err
	}
	if err := validateID(operationID, "operation_id"); err != nil {
		return authn.Principal{}, err
	}
	if len(requestID) > 64 {
		return authn.Principal{}, fmt.Errorf("%w: request_id exceeds 64 characters", ErrInvalid)
	}
	if operatorID == targetID {
		return authn.Principal{}, fmt.Errorf("%w: self authorization changes are not allowed", ErrForbidden)
	}

	var result authn.Principal
	err := m.store.WithinTransaction(ctx, func(tx TxStore) error {
		operation, exists, err := tx.FindOperation(ctx, operationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operatorID || operation.TargetAccountID != targetID {
				return fmt.Errorf("%w: operation_id was already used for a different change", ErrConflict)
			}
			result, err = tx.GetAuthorizationContext(ctx, targetID)
			return err
		}

		operator, err := tx.GetAuthorizationContext(ctx, operatorID)
		if err != nil {
			return err
		}
		if err := commonauthz.RequirePermission(operator, contractauthz.PermissionIdentityAuthorizationManage); err != nil {
			return fmt.Errorf("%w: %v", ErrForbidden, err)
		}
		before, err := tx.GetAuthorizationContext(ctx, targetID)
		if err != nil {
			return err
		}
		if err := mutate(tx, before); err != nil {
			return err
		}
		after, err := tx.GetAuthorizationContext(ctx, targetID)
		if err != nil {
			return err
		}
		if err := tx.RecordChange(ctx, Change{
			OperationID: operationID, OperatorAccountID: operatorID, TargetAccountID: targetID,
			Action: action, Before: before, After: after, RequestID: strings.TrimSpace(requestID),
		}); err != nil {
			return err
		}
		result = after
		return nil
	})
	if err != nil {
		if errors.Is(err, commonauthz.ErrPermissionDenied) || errors.Is(err, commonauthz.ErrUnauthenticated) {
			return authn.Principal{}, fmt.Errorf("%w: %v", ErrForbidden, err)
		}
		return authn.Principal{}, err
	}
	return result, nil
}

func validateID(value, field string) error {
	if _, err := uuid.Parse(strings.TrimSpace(value)); err != nil {
		return fmt.Errorf("%w: %s must be a UUID", ErrInvalid, field)
	}
	return nil
}
