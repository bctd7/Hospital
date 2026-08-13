package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
)

const (
	ActionExaminationItemCreated  = "appointment.catalog.examination_item.created"
	ActionExaminationItemUpdated  = "appointment.catalog.examination_item.updated"
	ActionExaminationItemDisabled = "appointment.catalog.examination_item.disabled"
	ActionExaminationItemEnabled  = "appointment.catalog.examination_item.enabled"
)

type Manager struct {
	store Store
}

func NewManager(store Store) (*Manager, error) {
	if store == nil {
		return nil, errors.New("examination catalog store is required")
	}
	return &Manager{store: store}, nil
}

func (m *Manager) Create(ctx context.Context, operator authn.Principal, command CreateCommand) (ExaminationItem, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentCreate); err != nil {
		return ExaminationItem{}, err
	}
	command, err := normalizedCreateCommand(command)
	if err != nil {
		return ExaminationItem{}, err
	}
	if err := requireDepartmentScope(operator, command.OwnerDepartmentID); err != nil {
		return ExaminationItem{}, err
	}
	fingerprint := operationFingerprint(ActionExaminationItemCreated, struct {
		OwnerDepartmentID string
		Name              string
		Description       string
	}{command.OwnerDepartmentID, command.Name, command.Description})

	var result ExaminationItem
	err = m.store.WithinCatalogTransaction(ctx, func(tx TxStore) error {
		operation, exists, err := tx.FindOperation(ctx, command.OperationID)
		if err != nil {
			return err
		}
		if exists {
			if err := matchingOperation(operation, operator.AccountID, "", ActionExaminationItemCreated, fingerprint); err != nil {
				return err
			}
			if err := requireDepartmentScope(operator, operation.Result.OwnerDepartmentID); err != nil {
				return err
			}
			result = operation.Result
			return nil
		}

		now := time.Now().UTC()
		result = ExaminationItem{
			ItemID:            uuid.NewString(),
			OwnerDepartmentID: command.OwnerDepartmentID,
			Name:              command.Name,
			Description:       command.Description,
			Status:            StatusActive,
			Version:           1,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		if err := tx.CreateItem(ctx, result); err != nil {
			return err
		}
		return tx.RecordChange(ctx, Change{
			OperationID:        command.OperationID,
			OperatorAccountID:  operator.AccountID,
			ItemID:             result.ItemID,
			Action:             ActionExaminationItemCreated,
			RequestFingerprint: fingerprint,
			After:              result,
			RequestID:          command.RequestID,
		})
	})
	if err != nil {
		return ExaminationItem{}, err
	}
	return result, nil
}

func (m *Manager) Get(ctx context.Context, operator authn.Principal, itemID string) (ExaminationItem, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return ExaminationItem{}, err
	}
	itemID, err := normalizedUUID(itemID, "item_id")
	if err != nil {
		return ExaminationItem{}, err
	}
	item, err := m.store.GetItem(ctx, itemID)
	if err != nil {
		return ExaminationItem{}, err
	}
	if err := requireDepartmentScope(operator, item.OwnerDepartmentID); err != nil {
		return ExaminationItem{}, err
	}
	return item, nil
}

func (m *Manager) List(ctx context.Context, operator authn.Principal, query ListQuery) (ListResult, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return ListResult{}, err
	}
	query, err := normalizedListQuery(query)
	if err != nil {
		return ListResult{}, err
	}
	query.OwnerDepartmentID, err = scopedListDepartment(operator, query.OwnerDepartmentID)
	if err != nil {
		return ListResult{}, err
	}
	offset, err := pageOffset(query.Page, query.PageSize)
	if err != nil {
		return ListResult{}, err
	}
	items, total, err := m.store.ListItems(ctx, ListFilter{
		OwnerDepartmentID: query.OwnerDepartmentID,
		Status:            query.Status,
		Offset:            offset,
		Limit:             query.PageSize,
	})
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (m *Manager) Update(ctx context.Context, operator authn.Principal, command UpdateCommand) (ExaminationItem, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ExaminationItem{}, err
	}
	command, err := normalizedUpdateCommand(command)
	if err != nil {
		return ExaminationItem{}, err
	}
	fingerprint := operationFingerprint(ActionExaminationItemUpdated, struct {
		ItemID          string
		Name            *string
		Description     *string
		ExpectedVersion int64
	}{command.ItemID, command.Name, command.Description, command.ExpectedVersion})

	var result ExaminationItem
	err = m.store.WithinCatalogTransaction(ctx, func(tx TxStore) error {
		operation, exists, err := tx.FindOperation(ctx, command.OperationID)
		if err != nil {
			return err
		}
		if exists {
			if err := matchingOperation(operation, operator.AccountID, command.ItemID, ActionExaminationItemUpdated, fingerprint); err != nil {
				return err
			}
			if err := requireDepartmentScope(operator, operation.Result.OwnerDepartmentID); err != nil {
				return err
			}
			result = operation.Result
			return nil
		}

		before, err := tx.GetItemForUpdate(ctx, command.ItemID)
		if err != nil {
			return err
		}
		if err := requireDepartmentScope(operator, before.OwnerDepartmentID); err != nil {
			return err
		}
		if before.Version != command.ExpectedVersion {
			return ErrVersionConflict
		}
		if before.Status != StatusActive {
			return fmt.Errorf("%w: disabled examination item must be enabled before update", ErrInvalidState)
		}

		after := before
		if command.Name != nil {
			after.Name = *command.Name
		}
		if command.Description != nil {
			after.Description = *command.Description
		}
		after.Version = before.Version + 1
		after.UpdatedAt = time.Now().UTC()
		if err := tx.UpdateItem(ctx, after, command.ExpectedVersion); err != nil {
			return err
		}
		if err := tx.RecordChange(ctx, Change{
			OperationID:        command.OperationID,
			OperatorAccountID:  operator.AccountID,
			ItemID:             command.ItemID,
			Action:             ActionExaminationItemUpdated,
			RequestFingerprint: fingerprint,
			Before:             &before,
			After:              after,
			RequestID:          command.RequestID,
		}); err != nil {
			return err
		}
		result = after
		return nil
	})
	if err != nil {
		return ExaminationItem{}, err
	}
	return result, nil
}

func (m *Manager) Disable(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (ExaminationItem, error) {
	return m.changeStatus(ctx, operator, command, StatusDisabled, ActionExaminationItemDisabled)
}

func (m *Manager) Enable(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (ExaminationItem, error) {
	return m.changeStatus(ctx, operator, command, StatusActive, ActionExaminationItemEnabled)
}

func (m *Manager) changeStatus(
	ctx context.Context,
	operator authn.Principal,
	command ChangeStatusCommand,
	target Status,
	action string,
) (ExaminationItem, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ExaminationItem{}, err
	}
	command, err := normalizedChangeStatusCommand(command)
	if err != nil {
		return ExaminationItem{}, err
	}
	fingerprint := operationFingerprint(action, struct {
		ItemID          string
		ExpectedVersion int64
	}{command.ItemID, command.ExpectedVersion})

	var result ExaminationItem
	err = m.store.WithinCatalogTransaction(ctx, func(tx TxStore) error {
		operation, exists, err := tx.FindOperation(ctx, command.OperationID)
		if err != nil {
			return err
		}
		if exists {
			if err := matchingOperation(operation, operator.AccountID, command.ItemID, action, fingerprint); err != nil {
				return err
			}
			if err := requireDepartmentScope(operator, operation.Result.OwnerDepartmentID); err != nil {
				return err
			}
			result = operation.Result
			return nil
		}

		before, err := tx.GetItemForUpdate(ctx, command.ItemID)
		if err != nil {
			return err
		}
		if err := requireDepartmentScope(operator, before.OwnerDepartmentID); err != nil {
			return err
		}
		if before.Version != command.ExpectedVersion {
			return ErrVersionConflict
		}

		after := before
		if before.Status != target {
			after.Status = target
			after.Version = before.Version + 1
			after.UpdatedAt = time.Now().UTC()
			if err := tx.SetItemStatus(ctx, after, command.ExpectedVersion); err != nil {
				return err
			}
		}
		if err := tx.RecordChange(ctx, Change{
			OperationID:        command.OperationID,
			OperatorAccountID:  operator.AccountID,
			ItemID:             command.ItemID,
			Action:             action,
			RequestFingerprint: fingerprint,
			Before:             &before,
			After:              after,
			RequestID:          command.RequestID,
		}); err != nil {
			return err
		}
		result = after
		return nil
	})
	if err != nil {
		return ExaminationItem{}, err
	}
	return result, nil
}

func requireCatalogPermission(operator authn.Principal, permission string) error {
	if err := commonauthz.RequirePermission(operator, permission); err != nil {
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	}
	if _, err := uuid.Parse(strings.TrimSpace(operator.AccountID)); err != nil {
		return fmt.Errorf("%w: invalid operator identity", ErrForbidden)
	}
	return nil
}

func requireDepartmentScope(operator authn.Principal, ownerDepartmentID string) error {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return nil
	}
	if !operator.HasRole(authn.RoleDepartmentDoctor) {
		return fmt.Errorf("%w: department staff role is required", ErrForbidden)
	}
	departmentID, err := normalizedUUID(operator.DepartmentID, "operator department_id")
	if err != nil || departmentID != ownerDepartmentID {
		return fmt.Errorf("%w: examination item belongs to another department", ErrForbidden)
	}
	return nil
}

func scopedListDepartment(operator authn.Principal, requested string) (string, error) {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return requested, nil
	}
	if !operator.HasRole(authn.RoleDepartmentDoctor) {
		return "", fmt.Errorf("%w: department staff role is required", ErrForbidden)
	}
	departmentID, err := normalizedUUID(operator.DepartmentID, "operator department_id")
	if err != nil {
		return "", fmt.Errorf("%w: invalid operator department", ErrForbidden)
	}
	if requested != "" && requested != departmentID {
		return "", fmt.Errorf("%w: examination catalog belongs to another department", ErrForbidden)
	}
	return departmentID, nil
}

func matchingOperation(operation Operation, operatorAccountID, itemID, action, fingerprint string) error {
	if operation.OperatorAccountID != operatorAccountID ||
		operation.Action != action ||
		operation.RequestFingerprint != fingerprint ||
		operation.ItemID == "" ||
		operation.Result.ItemID != operation.ItemID ||
		(itemID != "" && operation.ItemID != itemID) {
		return operationConflict()
	}
	return nil
}

func operationFingerprint(action string, payload any) string {
	data, err := json.Marshal(struct {
		Action  string
		Payload any
	}{action, payload})
	if err != nil {
		panic(fmt.Sprintf("marshal catalog operation fingerprint: %v", err))
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func operationConflict() error {
	return fmt.Errorf("%w: operation_id was already used for a different examination catalog change", ErrConflict)
}
