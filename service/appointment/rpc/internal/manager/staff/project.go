package staff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// Service implements project-domain rules and persistence coordination.
// Actor-facing use cases are exposed by internal/manager instead of this type.
func (m *Manager) CreateProject(ctx context.Context, operator authn.Principal, command CreateProjectCommand) (ExaminationItem, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentCreate); err != nil {
		return ExaminationItem{}, err
	}
	command, err := normalizedCreateProjectCommand(command)
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
	err = m.projectStore.WithinProjectTransaction(ctx, func(tx ProjectTxStore) error {
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
		return tx.RecordChange(ctx, ProjectChange{
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

func (m *Manager) GetProject(ctx context.Context, operator authn.Principal, itemID string) (ExaminationItem, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return ExaminationItem{}, err
	}
	itemID, err := normalizedUUID(itemID, "item_id")
	if err != nil {
		return ExaminationItem{}, err
	}
	item, err := m.projectStore.GetItem(ctx, itemID)
	if err != nil {
		return ExaminationItem{}, err
	}
	if err := requireDepartmentScope(operator, item.OwnerDepartmentID); err != nil {
		return ExaminationItem{}, err
	}
	return item, nil
}

func (m *Manager) ListProjects(ctx context.Context, operator authn.Principal, query ListProjectsQuery) (ListProjectsResult, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return ListProjectsResult{}, err
	}
	query, err := normalizedListProjectsQuery(query)
	if err != nil {
		return ListProjectsResult{}, err
	}
	query.OwnerDepartmentID, err = scopedListDepartment(operator, query.OwnerDepartmentID)
	if err != nil {
		return ListProjectsResult{}, err
	}
	offset, err := pageOffset(query.Page, query.PageSize)
	if err != nil {
		return ListProjectsResult{}, err
	}
	items, total, err := m.projectStore.ListItems(ctx, ProjectListFilter{
		OwnerDepartmentID: query.OwnerDepartmentID,
		Status:            query.Status,
		Offset:            offset,
		Limit:             query.PageSize,
	})
	if err != nil {
		return ListProjectsResult{}, err
	}
	return ListProjectsResult{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (m *Manager) UpdateProject(ctx context.Context, operator authn.Principal, command UpdateProjectCommand) (ExaminationItem, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ExaminationItem{}, err
	}
	command, err := normalizedUpdateProjectCommand(command)
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
	err = m.projectStore.WithinProjectTransaction(ctx, func(tx ProjectTxStore) error {
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
		if err := tx.RecordChange(ctx, ProjectChange{
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

func (m *Manager) DisableProject(ctx context.Context, operator authn.Principal, command ChangeProjectStatusCommand) (ExaminationItem, error) {
	return m.changeProjectStatus(ctx, operator, command, StatusDisabled, ActionExaminationItemDisabled)
}

func (m *Manager) EnableProject(ctx context.Context, operator authn.Principal, command ChangeProjectStatusCommand) (ExaminationItem, error) {
	return m.changeProjectStatus(ctx, operator, command, StatusActive, ActionExaminationItemEnabled)
}

func (m *Manager) changeProjectStatus(
	ctx context.Context,
	operator authn.Principal,
	command ChangeProjectStatusCommand,
	target Status,
	action string,
) (ExaminationItem, error) {
	if err := requireCatalogPermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ExaminationItem{}, err
	}
	command, err := normalizedChangeProjectStatusCommand(command)
	if err != nil {
		return ExaminationItem{}, err
	}
	fingerprint := operationFingerprint(action, struct {
		ItemID          string
		ExpectedVersion int64
	}{command.ItemID, command.ExpectedVersion})

	var result ExaminationItem
	err = m.projectStore.WithinProjectTransaction(ctx, func(tx ProjectTxStore) error {
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
			if target == StatusDisabled {
				today, weekEnd := currentBookingWeek()
				if _, err := deleteBookingsForConfiguration(ctx, tx, operator.AccountID, command.OperationID, BookingListFilter{
					ItemID: before.ItemID, FromDate: &today, ThroughDate: &weekEnd,
				}); err != nil {
					return err
				}
			}
			after.Status = target
			after.Version = before.Version + 1
			after.UpdatedAt = time.Now().UTC()
			if err := tx.SetItemStatus(ctx, after, command.ExpectedVersion); err != nil {
				return err
			}
		}
		if err := tx.RecordChange(ctx, ProjectChange{
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
	if target == StatusDisabled {
		m.invalidate(ctx, result.OwnerDepartmentID, "item-summary:"+result.ItemID)
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
		return "", fmt.Errorf("%w: examination project belongs to another department", ErrForbidden)
	}
	return departmentID, nil
}

func matchingOperation(operation ProjectOperation, operatorAccountID, itemID, action, fingerprint string) error {
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
		panic(fmt.Sprintf("marshal project operation fingerprint: %v", err))
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func operationConflict() error {
	return fmt.Errorf("%w: operation_id was already used for a different examination project change", ErrConflict)
}
