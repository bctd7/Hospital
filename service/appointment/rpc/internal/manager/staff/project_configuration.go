package staff

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	"hospital/service/appointment/rpc/internal/manager/common"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

const (
	projectConfigurationActionCreate   = "create"
	projectConfigurationActionUpdate   = "update"
	projectConfigurationStatePrepared  = "prepared"
	projectConfigurationStateConfirmed = "confirmed"
	projectConfigurationStateCancelled = "cancelled"
)

func (m *Manager) PrepareProjectConfiguration(ctx context.Context, operator authn.Principal, command staffinput.PrepareProjectConfiguration) (ProjectConfigurationTransaction, error) {
	if m.projectConfigurations == nil {
		return ProjectConfigurationTransaction{}, fmt.Errorf("%w: project configuration store is unavailable", ErrInvalidState)
	}
	action := strings.ToLower(strings.TrimSpace(command.Action))
	permission := contractauthz.PermissionAppointmentCreate
	if action == projectConfigurationActionUpdate {
		permission = contractauthz.PermissionAppointmentUpdate
	} else if action != projectConfigurationActionCreate {
		return ProjectConfigurationTransaction{}, ErrInvalid
	}
	if err := requireProjectPermission(operator, permission); err != nil {
		return ProjectConfigurationTransaction{}, err
	}
	transactionID, err := staffsupport.NormalizeUUID(command.TransactionID, "transaction_id")
	if err != nil {
		return ProjectConfigurationTransaction{}, err
	}
	itemID, err := staffsupport.NormalizeUUID(command.ItemID, "item_id")
	if err != nil {
		return ProjectConfigurationTransaction{}, err
	}
	departmentID, err := staffsupport.NormalizeUUID(command.OwnerDepartmentID, "owner_department_id")
	if err != nil {
		return ProjectConfigurationTransaction{}, err
	}
	if err := requireDepartmentScope(operator, departmentID); err != nil {
		return ProjectConfigurationTransaction{}, err
	}
	name, err := staffsupport.NormalizeProjectName(command.Name)
	if err != nil {
		return ProjectConfigurationTransaction{}, err
	}
	description, err := staffsupport.NormalizeProjectDescription(command.Description)
	if err != nil {
		return ProjectConfigurationTransaction{}, err
	}
	if command.EstimatedDurationMinutes < 5 || command.EstimatedDurationMinutes > 480 || command.EstimatedDurationMinutes%5 != 0 {
		return ProjectConfigurationTransaction{}, ErrInvalid
	}
	operationID, _, err := staffsupport.NormalizeOperation(command.OperationID, command.RequestID)
	if err != nil {
		return ProjectConfigurationTransaction{}, err
	}
	if (action == projectConfigurationActionCreate && command.ExpectedVersion != 0) ||
		(action == projectConfigurationActionUpdate && command.ExpectedVersion < 1) {
		return ProjectConfigurationTransaction{}, ErrInvalid
	}
	fingerprint := common.RequestFingerprint(struct {
		Action          string
		ItemID          string
		DepartmentID    string
		Name            string
		Description     string
		Duration        int32
		ExpectedVersion int64
	}{action, itemID, departmentID, name, description, command.EstimatedDurationMinutes, command.ExpectedVersion})

	var prepared ProjectConfigurationTransaction
	err = m.projectConfigurations.WithinProjectConfigurationTransaction(ctx, func(tx ProjectConfigurationTxStore) error {
		if existing, found, findErr := tx.GetConfigurationTransactionForUpdate(ctx, transactionID); findErr != nil {
			return findErr
		} else if found {
			if existing.OperatorAccountID != operator.AccountID || existing.RequestFingerprint != fingerprint || existing.OperationID != operationID {
				return ErrConflict
			}
			prepared = existing
			return nil
		}
		if existing, found, findErr := tx.FindConfigurationTransactionByOperation(ctx, operationID); findErr != nil {
			return findErr
		} else if found {
			if existing.TransactionID != transactionID || existing.OperatorAccountID != operator.AccountID || existing.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			prepared = existing
			return nil
		}
		if action == projectConfigurationActionUpdate {
			current, loadErr := tx.GetItemForUpdate(ctx, itemID)
			if loadErr != nil {
				return loadErr
			}
			if current.OwnerDepartmentID != departmentID || current.Version != command.ExpectedVersion || current.Status != StatusActive {
				return ErrVersionConflict
			}
		} else if _, loadErr := tx.GetItemForUpdate(ctx, itemID); loadErr == nil {
			return ErrConflict
		} else if !errors.Is(loadErr, ErrNotFound) {
			return loadErr
		}
		now := time.Now().UTC()
		prepared = ProjectConfigurationTransaction{
			TransactionID: transactionID, OperationID: operationID, OperatorAccountID: operator.AccountID,
			Action: action, ItemID: itemID, OwnerDepartmentID: departmentID, ItemName: name,
			Description:              description,
			EstimatedDurationMinutes: command.EstimatedDurationMinutes, ExpectedVersion: command.ExpectedVersion,
			State: projectConfigurationStatePrepared, RequestFingerprint: fingerprint, CreatedAt: now, UpdatedAt: now,
		}
		return tx.CreateConfigurationTransaction(ctx, prepared)
	})
	return prepared, err
}

func (m *Manager) ConfirmProjectConfiguration(ctx context.Context, operator authn.Principal, transactionID string) (ExaminationItem, error) {
	if m.projectConfigurations == nil {
		return ExaminationItem{}, ErrInvalidState
	}
	transactionID, err := staffsupport.NormalizeUUID(transactionID, "transaction_id")
	if err != nil {
		return ExaminationItem{}, err
	}
	var result ExaminationItem
	err = m.projectConfigurations.WithinProjectConfigurationTransaction(ctx, func(tx ProjectConfigurationTxStore) error {
		transaction, found, loadErr := tx.GetConfigurationTransactionForUpdate(ctx, transactionID)
		if loadErr != nil {
			return loadErr
		}
		if !found {
			return ErrNotFound
		}
		if transaction.OperatorAccountID != operator.AccountID && !operator.HasRole(authn.RoleSuperAdmin) {
			return ErrForbidden
		}
		if err := requireDepartmentScope(operator, transaction.OwnerDepartmentID); err != nil {
			return err
		}
		if transaction.State == projectConfigurationStateCancelled {
			return ErrInvalidState
		}
		if transaction.State == projectConfigurationStateConfirmed {
			item, itemErr := tx.GetItemForUpdate(ctx, transaction.ItemID)
			if itemErr != nil {
				return itemErr
			}
			result = item
			return nil
		}
		now := time.Now().UTC()
		if transaction.Action == projectConfigurationActionCreate {
			result = ExaminationItem{
				ItemID: transaction.ItemID, OwnerDepartmentID: transaction.OwnerDepartmentID,
				Name: transaction.ItemName, Description: transaction.Description, EstimatedDurationMinutes: transaction.EstimatedDurationMinutes,
				Status: StatusActive, Version: 1, CreatedAt: now, UpdatedAt: now,
			}
			if err := tx.CreateItem(ctx, result); err != nil {
				return err
			}
			if err := tx.RecordChange(ctx, ProjectChange{
				OperationID: transaction.OperationID, OperatorAccountID: transaction.OperatorAccountID,
				ItemID: result.ItemID, Action: ActionExaminationItemCreated,
				RequestFingerprint: transaction.RequestFingerprint, After: result,
			}); err != nil {
				return err
			}
		} else {
			before, loadErr := tx.GetItemForUpdate(ctx, transaction.ItemID)
			if loadErr != nil {
				return loadErr
			}
			if before.Version != transaction.ExpectedVersion || before.OwnerDepartmentID != transaction.OwnerDepartmentID || before.Status != StatusActive {
				return ErrVersionConflict
			}
			result = before
			result.Name = transaction.ItemName
			result.Description = transaction.Description
			result.EstimatedDurationMinutes = transaction.EstimatedDurationMinutes
			result.Version++
			result.UpdatedAt = now
			if err := tx.UpdateItem(ctx, result, transaction.ExpectedVersion); err != nil {
				return err
			}
			if err := tx.RecordChange(ctx, ProjectChange{
				OperationID: transaction.OperationID, OperatorAccountID: transaction.OperatorAccountID,
				ItemID: result.ItemID, Action: ActionExaminationItemUpdated,
				RequestFingerprint: transaction.RequestFingerprint, Before: &before, After: result,
			}); err != nil {
				return err
			}
		}
		return tx.SetConfigurationTransactionState(ctx, transactionID, projectConfigurationStateConfirmed, result.Version, now)
	})
	if err == nil {
		m.InvalidateItem(ctx, result.OwnerDepartmentID, result.ItemID)
	}
	return result, err
}

func (m *Manager) CancelProjectConfiguration(ctx context.Context, operator authn.Principal, transactionID string) (bool, error) {
	if m.projectConfigurations == nil {
		return false, ErrInvalidState
	}
	transactionID, err := staffsupport.NormalizeUUID(transactionID, "transaction_id")
	if err != nil {
		return false, err
	}
	cancelled := false
	err = m.projectConfigurations.WithinProjectConfigurationTransaction(ctx, func(tx ProjectConfigurationTxStore) error {
		transaction, found, loadErr := tx.GetConfigurationTransactionForUpdate(ctx, transactionID)
		if loadErr != nil {
			return loadErr
		}
		if !found {
			return ErrNotFound
		}
		if transaction.OperatorAccountID != operator.AccountID && !operator.HasRole(authn.RoleSuperAdmin) {
			return ErrForbidden
		}
		if transaction.State == projectConfigurationStateConfirmed {
			return ErrInvalidState
		}
		if transaction.State != projectConfigurationStateCancelled {
			if err := tx.SetConfigurationTransactionState(ctx, transactionID, projectConfigurationStateCancelled, 0, time.Now().UTC()); err != nil {
				return err
			}
		}
		cancelled = true
		return nil
	})
	return cancelled, err
}
