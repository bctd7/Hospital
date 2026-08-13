package manager

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/organization"
)

func (m *UnitManager) CreateUnit(ctx context.Context, operator authn.Principal, command CreateUnitCommand) (organization.Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return organization.Unit{}, err
	}
	if command.Type != organization.UnitTypeCampus && command.Type != organization.UnitTypeDepartment {
		return organization.Unit{}, fmt.Errorf("%w: only campus and department can be created", organization.ErrInvalid)
	}
	parentID, err := normalizedUUID(command.ParentID, "parent_id")
	if err != nil {
		return organization.Unit{}, err
	}
	name, err := normalizedUnitName(command.Name)
	if err != nil {
		return organization.Unit{}, err
	}
	operationID, requestID, err := normalizedOperation(command.OperationID, command.RequestID)
	if err != nil {
		return organization.Unit{}, err
	}

	var result organization.Unit
	err = m.store.WithinOrganizationTransaction(ctx, func(tx organization.TxStore) error {
		operation, exists, err := tx.FindOperation(ctx, operationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operator.AccountID ||
				operation.Action != ActionUnitCreated ||
				operation.Result.Type != command.Type ||
				operation.Result.ParentID != parentID ||
				operation.Result.Name != name {
				return operationConflict()
			}
			result = operation.Result
			return nil
		}

		parent, err := tx.GetUnitForUpdate(ctx, parentID)
		if err != nil {
			return err
		}
		if err := requireActiveParent(parent, command.Type); err != nil {
			return err
		}

		result = organization.Unit{
			ID:       uuid.NewString(),
			ParentID: parentID,
			Type:     command.Type,
			Code:     generatedUnitCode(command.Type, operationID),
			Name:     name,
			Status:   organization.StatusActive,
			Version:  1,
		}
		if err := tx.CreateUnit(ctx, result); err != nil {
			return err
		}
		return tx.RecordChange(ctx, organization.Change{
			OperationID:       operationID,
			OperatorAccountID: operator.AccountID,
			UnitID:            result.ID,
			Action:            ActionUnitCreated,
			After:             result,
			RequestID:         requestID,
		})
	})
	if err != nil {
		return organization.Unit{}, err
	}
	return result, nil
}

func (m *UnitManager) UpdateUnit(ctx context.Context, operator authn.Principal, command UpdateUnitCommand) (organization.Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return organization.Unit{}, err
	}
	unitID, err := normalizedUUID(command.UnitID, "unit_id")
	if err != nil {
		return organization.Unit{}, err
	}
	if command.ExpectedVersion <= 0 {
		return organization.Unit{}, fmt.Errorf("%w: version must be positive", organization.ErrInvalid)
	}
	if command.Name == nil && command.ParentID == nil {
		return organization.Unit{}, fmt.Errorf("%w: name or parent_id is required", organization.ErrInvalid)
	}

	var name *string
	if command.Name != nil {
		normalized, err := normalizedUnitName(*command.Name)
		if err != nil {
			return organization.Unit{}, err
		}
		name = &normalized
	}
	var parentID *string
	if command.ParentID != nil {
		normalized, err := normalizedUUID(*command.ParentID, "parent_id")
		if err != nil {
			return organization.Unit{}, err
		}
		parentID = &normalized
	}
	operationID, requestID, err := normalizedOperation(command.OperationID, command.RequestID)
	if err != nil {
		return organization.Unit{}, err
	}

	var result organization.Unit
	err = m.store.WithinOrganizationTransaction(ctx, func(tx organization.TxStore) error {
		operation, exists, err := tx.FindOperation(ctx, operationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operator.AccountID ||
				operation.UnitID != unitID || operation.Action != ActionUnitUpdated ||
				(name != nil && operation.Result.Name != *name) ||
				(parentID != nil && operation.Result.ParentID != *parentID) {
				return operationConflict()
			}
			result = operation.Result
			return nil
		}

		before, err := tx.GetUnitForUpdate(ctx, unitID)
		if err != nil {
			return err
		}
		if before.Type == organization.UnitTypeHospital {
			return fmt.Errorf("%w: hospital root is read-only", organization.ErrForbidden)
		}
		if before.Version != command.ExpectedVersion {
			return organization.ErrVersionConflict
		}

		after := before
		if name != nil {
			after.Name = *name
		}
		if parentID != nil && *parentID != before.ParentID {
			if before.Type != organization.UnitTypeDepartment {
				return fmt.Errorf("%w: campus parent cannot be changed", organization.ErrInvalidHierarchy)
			}
			parent, err := tx.GetUnitForUpdate(ctx, *parentID)
			if err != nil {
				return err
			}
			if err := requireActiveParent(parent, before.Type); err != nil {
				return err
			}
			after.ParentID = *parentID
		}
		after.Version = before.Version + 1

		if err := tx.UpdateUnit(ctx, after, command.ExpectedVersion); err != nil {
			return err
		}
		if err := tx.RecordChange(ctx, organization.Change{
			OperationID:       operationID,
			OperatorAccountID: operator.AccountID,
			UnitID:            unitID,
			Action:            ActionUnitUpdated,
			Before:            &before,
			After:             after,
			RequestID:         requestID,
		}); err != nil {
			return err
		}
		result = after
		return nil
	})
	if err != nil {
		return organization.Unit{}, err
	}
	return result, nil
}

func (m *UnitManager) DisableUnit(ctx context.Context, operator authn.Principal, command ChangeUnitStatusCommand) (organization.Unit, error) {
	return m.changeStatus(ctx, operator, command, organization.StatusDisabled, ActionUnitDisabled)
}

func (m *UnitManager) EnableUnit(ctx context.Context, operator authn.Principal, command ChangeUnitStatusCommand) (organization.Unit, error) {
	return m.changeStatus(ctx, operator, command, organization.StatusActive, ActionUnitEnabled)
}

func (m *UnitManager) changeStatus(
	ctx context.Context,
	operator authn.Principal,
	command ChangeUnitStatusCommand,
	status organization.Status,
	action string,
) (organization.Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return organization.Unit{}, err
	}
	unitID, err := normalizedUUID(command.UnitID, "unit_id")
	if err != nil {
		return organization.Unit{}, err
	}
	if command.ExpectedVersion <= 0 {
		return organization.Unit{}, fmt.Errorf("%w: version must be positive", organization.ErrInvalid)
	}
	operationID, requestID, err := normalizedOperation(command.OperationID, command.RequestID)
	if err != nil {
		return organization.Unit{}, err
	}

	var result organization.Unit
	err = m.store.WithinOrganizationTransaction(ctx, func(tx organization.TxStore) error {
		operation, exists, err := tx.FindOperation(ctx, operationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operator.AccountID ||
				operation.UnitID != unitID || operation.Action != action {
				return operationConflict()
			}
			result = operation.Result
			return nil
		}

		before, err := tx.GetUnitForUpdate(ctx, unitID)
		if err != nil {
			return err
		}
		if before.Type == organization.UnitTypeHospital {
			return fmt.Errorf("%w: hospital root is read-only", organization.ErrForbidden)
		}
		if before.Version != command.ExpectedVersion {
			return organization.ErrVersionConflict
		}

		if before.Status == status {
			result = before
			return tx.RecordChange(ctx, organization.Change{
				OperationID:       operationID,
				OperatorAccountID: operator.AccountID,
				UnitID:            unitID,
				Action:            action,
				Before:            &before,
				After:             before,
				RequestID:         requestID,
			})
		}

		if status == organization.StatusDisabled {
			if err := ensureUnitCanBeDisabled(ctx, tx, before); err != nil {
				return err
			}
		} else {
			parent, err := tx.GetUnitForUpdate(ctx, before.ParentID)
			if err != nil {
				return err
			}
			if err := requireActiveParent(parent, before.Type); err != nil {
				return err
			}
		}

		after := before
		after.Status = status
		after.Version = before.Version + 1
		if err := tx.SetUnitStatus(ctx, unitID, status, command.ExpectedVersion); err != nil {
			return err
		}
		if err := tx.RecordChange(ctx, organization.Change{
			OperationID:       operationID,
			OperatorAccountID: operator.AccountID,
			UnitID:            unitID,
			Action:            action,
			Before:            &before,
			After:             after,
			RequestID:         requestID,
		}); err != nil {
			return err
		}
		result = after
		return nil
	})
	if err != nil {
		return organization.Unit{}, err
	}
	return result, nil
}
