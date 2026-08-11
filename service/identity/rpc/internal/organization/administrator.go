package organization

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
)

func (m *Manager) GetManagedUnit(
	ctx context.Context,
	operator authn.Principal,
	unitID string,
) (Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return Unit{}, err
	}

	unit, err := m.getUnit(ctx, unitID)
	if err != nil {
		return Unit{}, err
	}
	if unit.Type == UnitTypeHospital {
		return Unit{}, fmt.Errorf("%w: hospital root is read-only", ErrForbidden)
	}
	return unit, nil
}

// ListManagedUnits applies the administrator permission and hierarchy rules
// before delegating the flat query to Store.
func (m *Manager) ListManagedUnits(
	ctx context.Context,
	operator authn.Principal,
	filter ListFilter,
) ([]Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return nil, err
	}
	if filter.Type != UnitTypeCampus && filter.Type != UnitTypeDepartment {
		return nil, fmt.Errorf("%w: unit_type must be campus or department", ErrInvalid)
	}
	if filter.ParentID == nil {
		return nil, fmt.Errorf("%w: parent_id is required", ErrInvalid)
	}
	parentID, err := normalizedUUID(*filter.ParentID, "parent_id")
	if err != nil {
		return nil, err
	}
	filter.ParentID = &parentID
	if filter.Status != nil && !filter.Status.Valid() {
		return nil, fmt.Errorf("%w: unsupported status %q", ErrInvalid, *filter.Status)
	}

	parent, err := m.store.GetUnit(ctx, parentID)
	if err != nil {
		return nil, err
	}
	requiredParentType, _ := filter.Type.RequiredParentType()
	if parent.Type != requiredParentType {
		return nil, fmt.Errorf("%w: %s requires a %s parent", ErrInvalidHierarchy, filter.Type, requiredParentType)
	}
	return m.store.ListUnits(ctx, filter)
}

func (m *Manager) CreateUnit(ctx context.Context, operator authn.Principal, command CreateUnitCommand) (Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return Unit{}, err
	}
	if command.Type != UnitTypeCampus && command.Type != UnitTypeDepartment {
		return Unit{}, fmt.Errorf("%w: only campus and department can be created", ErrInvalid)
	}
	parentID, err := normalizedUUID(command.ParentID, "parent_id")
	if err != nil {
		return Unit{}, err
	}
	name, err := normalizedUnitName(command.Name)
	if err != nil {
		return Unit{}, err
	}
	operationID, requestID, err := normalizedOperation(command.OperationID, command.RequestID)
	if err != nil {
		return Unit{}, err
	}

	var result Unit
	err = m.store.WithinOrganizationTransaction(ctx, func(tx TxStore) error {
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

		result = Unit{
			ID:       uuid.NewString(),
			ParentID: parentID,
			Type:     command.Type,
			Code:     generatedUnitCode(command.Type, operationID),
			Name:     name,
			Status:   StatusActive,
			Version:  1,
		}
		if err := tx.CreateUnit(ctx, result); err != nil {
			return err
		}
		return tx.RecordChange(ctx, Change{
			OperationID:       operationID,
			OperatorAccountID: operator.AccountID,
			UnitID:            result.ID,
			Action:            ActionUnitCreated,
			After:             result,
			RequestID:         requestID,
		})
	})
	if err != nil {
		return Unit{}, err
	}
	return result, nil
}

func (m *Manager) UpdateUnit(ctx context.Context, operator authn.Principal, command UpdateUnitCommand) (Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return Unit{}, err
	}
	unitID, err := normalizedUUID(command.UnitID, "unit_id")
	if err != nil {
		return Unit{}, err
	}
	if command.ExpectedVersion <= 0 {
		return Unit{}, fmt.Errorf("%w: version must be positive", ErrInvalid)
	}
	if command.Name == nil && command.ParentID == nil {
		return Unit{}, fmt.Errorf("%w: name or parent_id is required", ErrInvalid)
	}

	var name *string
	if command.Name != nil {
		normalized, err := normalizedUnitName(*command.Name)
		if err != nil {
			return Unit{}, err
		}
		name = &normalized
	}
	var parentID *string
	if command.ParentID != nil {
		normalized, err := normalizedUUID(*command.ParentID, "parent_id")
		if err != nil {
			return Unit{}, err
		}
		parentID = &normalized
	}
	operationID, requestID, err := normalizedOperation(command.OperationID, command.RequestID)
	if err != nil {
		return Unit{}, err
	}

	var result Unit
	err = m.store.WithinOrganizationTransaction(ctx, func(tx TxStore) error {
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
		if before.Type == UnitTypeHospital {
			return fmt.Errorf("%w: hospital root is read-only", ErrForbidden)
		}
		if before.Version != command.ExpectedVersion {
			return ErrVersionConflict
		}

		after := before
		if name != nil {
			after.Name = *name
		}
		if parentID != nil && *parentID != before.ParentID {
			if before.Type != UnitTypeDepartment {
				return fmt.Errorf("%w: campus parent cannot be changed", ErrInvalidHierarchy)
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
		if err := tx.RecordChange(ctx, Change{
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
		return Unit{}, err
	}
	return result, nil
}

func (m *Manager) DisableUnit(ctx context.Context, operator authn.Principal, command ChangeUnitStatusCommand) (Unit, error) {
	return m.changeStatus(ctx, operator, command, StatusDisabled, ActionUnitDisabled)
}

func (m *Manager) EnableUnit(ctx context.Context, operator authn.Principal, command ChangeUnitStatusCommand) (Unit, error) {
	return m.changeStatus(ctx, operator, command, StatusActive, ActionUnitEnabled)
}

func (m *Manager) changeStatus(
	ctx context.Context,
	operator authn.Principal,
	command ChangeUnitStatusCommand,
	status Status,
	action string,
) (Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return Unit{}, err
	}
	unitID, err := normalizedUUID(command.UnitID, "unit_id")
	if err != nil {
		return Unit{}, err
	}
	if command.ExpectedVersion <= 0 {
		return Unit{}, fmt.Errorf("%w: version must be positive", ErrInvalid)
	}
	operationID, requestID, err := normalizedOperation(command.OperationID, command.RequestID)
	if err != nil {
		return Unit{}, err
	}

	var result Unit
	err = m.store.WithinOrganizationTransaction(ctx, func(tx TxStore) error {
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
		if before.Type == UnitTypeHospital {
			return fmt.Errorf("%w: hospital root is read-only", ErrForbidden)
		}
		if before.Version != command.ExpectedVersion {
			return ErrVersionConflict
		}

		if before.Status == status {
			result = before
			return tx.RecordChange(ctx, Change{
				OperationID:       operationID,
				OperatorAccountID: operator.AccountID,
				UnitID:            unitID,
				Action:            action,
				Before:            &before,
				After:             before,
				RequestID:         requestID,
			})
		}

		if status == StatusDisabled {
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
		if err := tx.RecordChange(ctx, Change{
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
		return Unit{}, err
	}
	return result, nil
}

func requireManagePermission(operator authn.Principal) error {
	if err := commonauthz.RequirePermission(operator, contractauthz.PermissionIdentityDepartmentManage); err != nil {
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	}
	if _, err := uuid.Parse(operator.AccountID); err != nil {
		return fmt.Errorf("%w: invalid operator identity", ErrForbidden)
	}
	return nil
}

func requireActiveParent(parent Unit, childType UnitType) error {
	requiredType, hasParent := childType.RequiredParentType()
	if !hasParent || parent.Type != requiredType {
		return fmt.Errorf("%w: %s requires an active %s parent", ErrInvalidHierarchy, childType, requiredType)
	}
	if parent.Status != StatusActive {
		return fmt.Errorf("%w: parent organization unit is disabled", ErrInvalidHierarchy)
	}
	return nil
}

func ensureUnitCanBeDisabled(ctx context.Context, tx TxStore, unit Unit) error {
	switch unit.Type {
	case UnitTypeCampus:
		count, err := tx.CountActiveChildren(ctx, unit.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrActiveChildren
		}
	case UnitTypeDepartment:
		count, err := tx.CountActiveDoctors(ctx, unit.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrActiveDoctors
		}
	default:
		return fmt.Errorf("%w: unsupported unit type %q", ErrInvalid, unit.Type)
	}
	return nil
}
