package organization

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
)

const (
	ActionUnitCreated  = "identity.organization.unit.created"
	ActionUnitUpdated  = "identity.organization.unit.updated"
	ActionUnitDisabled = "identity.organization.unit.disabled"
	ActionUnitEnabled  = "identity.organization.unit.enabled"

	maxUnitNameRunes  = 128
	maxRequestIDBytes = 64
)

type CreateUnitCommand struct {
	Type        UnitType
	ParentID    string
	Name        string
	OperationID string
	RequestID   string
}

type UpdateUnitCommand struct {
	UnitID          string
	Name            *string
	ParentID        *string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type ChangeUnitStatusCommand struct {
	UnitID          string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type Manager struct {
	store Store
}

func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

func (m *Manager) GetUnit(ctx context.Context, unitID string) (Unit, error) {
	unitID, err := normalizedUUID(unitID, "unit_id")
	if err != nil {
		return Unit{}, err
	}
	return m.store.GetUnit(ctx, unitID)
}

func (m *Manager) GetManagedUnit(
	ctx context.Context,
	operator authn.Principal,
	unitID string,
) (Unit, error) {
	if err := requireManagePermission(operator); err != nil {
		return Unit{}, err
	}

	unit, err := m.GetUnit(ctx, unitID)
	if err != nil {
		return Unit{}, err
	}
	if unit.Type == UnitTypeHospital {
		return Unit{}, fmt.Errorf("%w: hospital root is read-only", ErrForbidden)
	}
	return unit, nil
}

func (m *Manager) ListUnits(ctx context.Context, filter ListFilter) ([]Unit, error) {
	if !filter.Type.Valid() {
		return nil, fmt.Errorf("%w: unsupported unit_type %q", ErrInvalid, filter.Type)
	}
	if filter.ParentID != nil {
		parentID, err := normalizedUUID(*filter.ParentID, "parent_id")
		if err != nil {
			return nil, err
		}
		filter.ParentID = &parentID
	}
	if filter.Status != nil && !filter.Status.Valid() {
		return nil, fmt.Errorf("%w: unsupported status %q", ErrInvalid, *filter.Status)
	}
	return m.store.ListUnits(ctx, filter)
}

// GetDirectoryContext returns the public organization entry point. It does
// not require an operator because the plan explicitly exposes this directory
// to unauthenticated visitors as well as authenticated users.
func (m *Manager) GetDirectoryContext(ctx context.Context) (DirectoryContext, error) {
	active := StatusActive
	hospitals, err := m.ListUnits(ctx, ListFilter{Type: UnitTypeHospital, Status: &active})
	if err != nil {
		return DirectoryContext{}, err
	}
	if len(hospitals) != 1 {
		return DirectoryContext{}, fmt.Errorf("%w: expected one active hospital root", ErrDirectoryUnavailable)
	}

	hospital := hospitals[0]
	campuses, err := m.ListUnits(ctx, ListFilter{
		Type:     UnitTypeCampus,
		ParentID: &hospital.ID,
		Status:   &active,
	})
	if err != nil {
		return DirectoryContext{}, err
	}
	return DirectoryContext{Hospital: hospital, Campuses: campuses}, nil
}

// ListDirectoryDepartments returns active departments under one active
// campus. Disabled or non-campus parents are hidden as not found so public
// callers cannot enumerate inactive organization structure.
func (m *Manager) ListDirectoryDepartments(ctx context.Context, campusID string) ([]DirectoryDepartment, error) {
	campusID, err := normalizedUUID(campusID, "campus_id")
	if err != nil {
		return nil, err
	}
	campus, err := m.store.GetUnit(ctx, campusID)
	if err != nil {
		return nil, err
	}
	if campus.Type != UnitTypeCampus || campus.Status != StatusActive {
		return nil, ErrNotFound
	}

	active := StatusActive
	units, err := m.ListUnits(ctx, ListFilter{
		Type:     UnitTypeDepartment,
		ParentID: &campusID,
		Status:   &active,
	})
	if err != nil {
		return nil, err
	}
	departments := make([]DirectoryDepartment, 0, len(units))
	for _, unit := range units {
		departments = append(departments, DirectoryDepartment{Unit: unit, CampusName: campus.Name})
	}
	return departments, nil
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

func normalizedUnitName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if utf8.RuneCountInString(value) > maxUnitNameRunes {
		return "", fmt.Errorf("%w: name exceeds %d characters", ErrInvalid, maxUnitNameRunes)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: name contains control characters", ErrInvalid)
	}
	return value, nil
}

func normalizedUUID(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := uuid.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", ErrInvalid, field)
	}
	return parsed.String(), nil
}

func normalizedOperation(operationID, requestID string) (string, string, error) {
	operationID, err := normalizedUUID(operationID, "operation_id")
	if err != nil {
		return "", "", err
	}
	requestID = strings.TrimSpace(requestID)
	if len(requestID) > maxRequestIDBytes {
		return "", "", fmt.Errorf("%w: request_id exceeds %d bytes", ErrInvalid, maxRequestIDBytes)
	}
	return operationID, requestID, nil
}

func generatedUnitCode(unitType UnitType, operationID string) string {
	prefix := "CAMPUS"
	if unitType == UnitTypeDepartment {
		prefix = "DEPT"
	}
	compactID := strings.ReplaceAll(operationID, "-", "")
	return prefix + "-" + strings.ToUpper(compactID[:12])
}

func operationConflict() error {
	return fmt.Errorf("%w: operation_id was already used for a different organization change", ErrConflict)
}
