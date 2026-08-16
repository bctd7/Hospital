package staff

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
	"hospital/service/appointment/rpc/internal/manager/common"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

const (
	ActionExaminationItemCreated             = "appointment.project.examination_item.created"
	ActionExaminationItemUpdated             = "appointment.project.examination_item.updated"
	ActionExaminationItemDisabled            = "appointment.project.examination_item.disabled"
	ActionExaminationItemEnabled             = "appointment.project.examination_item.enabled"
	ActionExaminationItemReportTemplateSaved = "appointment.project.examination_item.report_template.saved"
	actionItemWindowSet                      = "appointment.resource.item_window.set"
	actionItemWindowDisabled                 = "appointment.resource.item_window.disabled"
)

// SaveProjectReportTemplate 更新项目级报告默认正文，不改写已有草稿或正式报告。
func (m *Manager) SaveProjectReportTemplate(ctx context.Context, operator authn.Principal, command staffinput.SaveReportTemplate) (ExaminationItem, error) {
	if err := requireProjectPermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ExaminationItem{}, err
	}
	itemID, err := staffsupport.NormalizeUUID(command.ItemID, "item_id")
	if err != nil {
		return ExaminationItem{}, err
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return ExaminationItem{}, err
	}
	template, err := normalizeReportContent(command.Template, false)
	if err != nil {
		return ExaminationItem{}, err
	}
	if command.ExpectedTemplateVersion < 0 {
		return ExaminationItem{}, fmt.Errorf("%w: expected_template_version cannot be negative", ErrInvalid)
	}
	fingerprint := common.RequestFingerprint(struct {
		ItemID                  string
		Template                ReportContent
		ExpectedTemplateVersion int64
	}{itemID, template, command.ExpectedTemplateVersion})

	var result ExaminationItem
	err = m.projectWrites.WithinProjectTransaction(ctx, func(tx ProjectTxStore) error {
		operation, exists, findErr := tx.FindOperation(ctx, meta.OperationID)
		if findErr != nil {
			return findErr
		}
		if exists {
			if matchErr := matchingOperation(operation, operator.AccountID, itemID, ActionExaminationItemReportTemplateSaved, fingerprint); matchErr != nil {
				return matchErr
			}
			if scopeErr := requireDepartmentScope(operator, operation.Result.OwnerDepartmentID); scopeErr != nil {
				return scopeErr
			}
			result = operation.Result
			return nil
		}
		before, lockErr := tx.GetItemForUpdate(ctx, itemID)
		if lockErr != nil {
			return lockErr
		}
		if scopeErr := requireDepartmentScope(operator, before.OwnerDepartmentID); scopeErr != nil {
			return scopeErr
		}
		if before.ReportTemplate.Version != command.ExpectedTemplateVersion {
			return ErrVersionConflict
		}
		after := before
		after.ReportTemplate = ReportTemplate{ReportContent: template, Version: before.ReportTemplate.Version + 1}
		after.UpdatedAt = time.Now().UTC()
		if updateErr := tx.UpdateItemReportTemplate(ctx, after, command.ExpectedTemplateVersion); updateErr != nil {
			return updateErr
		}
		if recordErr := tx.RecordChange(ctx, ProjectChange{
			OperationID: meta.OperationID, OperatorAccountID: operator.AccountID,
			ItemID: itemID, Action: ActionExaminationItemReportTemplateSaved,
			RequestFingerprint: fingerprint, Before: &before, After: after, RequestID: meta.RequestID,
		}); recordErr != nil {
			return recordErr
		}
		result = after
		return nil
	})
	return result, err
}

// CreateProject 编排项目创建规则、幂等检查、事务写入和审计记录。
func (m *Manager) CreateProject(ctx context.Context, operator authn.Principal, command staffinput.CreateProject) (ExaminationItem, error) {
	if err := requireProjectPermission(operator, contractauthz.PermissionAppointmentCreate); err != nil {
		return ExaminationItem{}, err
	}
	command, err := staffinput.NormalizeCreateProject(command)
	if err != nil {
		return ExaminationItem{}, err
	}
	if err := requireDepartmentScope(operator, command.OwnerDepartmentID); err != nil {
		return ExaminationItem{}, err
	}
	fingerprint := common.RequestFingerprint(struct {
		OwnerDepartmentID        string
		Name                     string
		Description              string
		EstimatedDurationMinutes int32
	}{command.OwnerDepartmentID, command.Name, command.Description, command.EstimatedDurationMinutes})

	var result ExaminationItem
	err = m.projectWrites.WithinProjectTransaction(ctx, func(tx ProjectTxStore) error {
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
			ItemID:                   uuid.NewString(),
			OwnerDepartmentID:        command.OwnerDepartmentID,
			Name:                     command.Name,
			Description:              command.Description,
			EstimatedDurationMinutes: command.EstimatedDurationMinutes,
			Status:                   StatusActive,
			Version:                  1,
			CreatedAt:                now,
			UpdatedAt:                now,
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

// UpdateProject 修改项目名称或检查说明，并执行乐观锁校验。
func (m *Manager) UpdateProject(ctx context.Context, operator authn.Principal, command staffinput.UpdateProject) (ExaminationItem, error) {
	if err := requireProjectPermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ExaminationItem{}, err
	}
	command, err := staffinput.NormalizeUpdateProject(command)
	if err != nil {
		return ExaminationItem{}, err
	}
	fingerprint := common.RequestFingerprint(struct {
		ItemID                   string
		Name                     *string
		Description              *string
		EstimatedDurationMinutes *int32
		ExpectedVersion          int64
	}{command.ItemID, command.Name, command.Description, command.EstimatedDurationMinutes, command.ExpectedVersion})

	var result ExaminationItem
	err = m.projectWrites.WithinProjectTransaction(ctx, func(tx ProjectTxStore) error {
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
		if command.EstimatedDurationMinutes != nil {
			after.EstimatedDurationMinutes = *command.EstimatedDurationMinutes
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

// DisableProject 停用检查项目并处理当前周受影响的预约。
func (m *Manager) DisableProject(ctx context.Context, operator authn.Principal, command staffinput.ChangeProjectStatus) (ExaminationItem, error) {
	return m.changeProjectStatus(ctx, operator, command, StatusDisabled, ActionExaminationItemDisabled)
}

// EnableProject 重新启用检查项目。
func (m *Manager) EnableProject(ctx context.Context, operator authn.Principal, command staffinput.ChangeProjectStatus) (ExaminationItem, error) {
	return m.changeProjectStatus(ctx, operator, command, StatusActive, ActionExaminationItemEnabled)
}

func (m *Manager) changeProjectStatus(
	ctx context.Context,
	operator authn.Principal,
	command staffinput.ChangeProjectStatus,
	target Status,
	action string,
) (ExaminationItem, error) {
	if err := requireProjectPermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ExaminationItem{}, err
	}
	command, err := staffinput.NormalizeChangeProjectStatus(command)
	if err != nil {
		return ExaminationItem{}, err
	}
	fingerprint := common.RequestFingerprint(struct {
		ItemID          string
		ExpectedVersion int64
	}{command.ItemID, command.ExpectedVersion})

	var result ExaminationItem
	err = m.projectWrites.WithinProjectTransaction(ctx, func(tx ProjectTxStore) error {
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
				if _, err := deleteBookingsForConfiguration(ctx, tx, operator.AccountID, BookingListFilter{
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

func requireProjectPermission(operator authn.Principal, permission string) error {
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
	departmentID, err := staffsupport.NormalizeUUID(operator.DepartmentID, "operator department_id")
	if err != nil || departmentID != ownerDepartmentID {
		return fmt.Errorf("%w: examination item belongs to another department", ErrForbidden)
	}
	return nil
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

func operationConflict() error {
	return fmt.Errorf("%w: operation_id was already used for a different examination project change", ErrConflict)
}

// SetItemWindow 创建或修改检查项目自己的预约时间和截止时间。
func (m *Manager) SetItemWindow(ctx context.Context, operator authn.Principal, command staffinput.SetItemWindow) (ItemWeeklyWindow, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ItemWeeklyWindow{}, err
	}
	itemID, err := staffsupport.NormalizeUUID(command.ItemID, "item_id")
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	windowID := ""
	if strings.TrimSpace(command.WindowID) != "" {
		windowID, err = staffsupport.NormalizeUUID(command.WindowID, "window_id")
		if err != nil {
			return ItemWeeklyWindow{}, err
		}
	}
	weekday, session, err := staffsupport.NormalizeSlot(command.Weekday, command.Session)
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	start, startMin, err := staffsupport.NormalizeClock(command.StartTime, "start_time")
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	cutoff, cutoffMin, err := staffsupport.NormalizeClock(command.BookingCutoffTime, "booking_cutoff_time")
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	end, endMin, err := staffsupport.NormalizeClock(command.EndTime, "end_time")
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	if startMin > cutoffMin || cutoffMin >= endMin || !staffsupport.FitsSessionBoundary(session, startMin, endMin) {
		return ItemWeeklyWindow{}, fmt.Errorf("%w: item window time is invalid", ErrInvalid)
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	if windowID == "" && command.ExpectedVersion != 0 {
		return ItemWeeklyWindow{}, fmt.Errorf("%w: expected_version must be zero for a new window", ErrInvalid)
	}
	if windowID != "" && command.ExpectedVersion < 1 {
		return ItemWeeklyWindow{}, fmt.Errorf("%w: expected_version is required", ErrInvalid)
	}
	command.WindowID, command.ItemID, command.Weekday, command.Session, command.StartTime, command.BookingCutoffTime, command.EndTime, command.Operation = windowID, itemID, weekday, session, start, cutoff, end, meta
	var result ItemWeeklyWindow
	var departmentID string
	err = m.mutate(ctx, m.projects, operator, meta, "item_window", windowID, actionItemWindowSet, command, func() string { return departmentID }, func(tx ConfigurationTxStore) (any, string, error) {
		item, lockErr := tx.GetItemForUpdate(ctx, itemID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = item.DepartmentID
		if err := staffsupport.RequireDepartmentScope(operator, departmentID); err != nil {
			return nil, "", err
		}
		if item.Status != StatusActive {
			return nil, "", ErrInvalidState
		}
		now := time.Now().UTC()
		var before *ItemWeeklyWindow
		if windowID == "" {
			if _, exists, e := tx.FindItemWindowForUpdate(ctx, itemID, weekday, session); e != nil {
				return nil, "", e
			} else if exists {
				return nil, "", fmt.Errorf("%w: item window slot already exists", ErrConflict)
			}
			result = ItemWeeklyWindow{WindowID: uuid.NewString(), ItemID: itemID, Weekday: weekday, Session: session, StartTime: start, BookingCutoffTime: cutoff, EndTime: end, Status: StatusActive, Version: 1, CreatedAt: now, UpdatedAt: now}
		} else {
			current, e := tx.GetItemWindowForUpdate(ctx, windowID)
			if e != nil {
				return nil, "", e
			}
			if current.ItemID != itemID || current.Weekday != weekday || current.Session != session {
				return nil, "", ErrConflict
			}
			if current.Version != command.ExpectedVersion {
				return nil, "", ErrVersionConflict
			}
			before = &current
			result = current
			result.StartTime, result.BookingCutoffTime, result.EndTime, result.Status, result.Version, result.UpdatedAt = start, cutoff, end, StatusActive, current.Version+1, now
		}
		if err := staffsupport.ValidateItemWindowChange(ctx, tx, result); err != nil {
			return nil, "", err
		}
		if before == nil {
			if err := tx.CreateItemWindow(ctx, result); err != nil {
				return nil, "", err
			}
		} else {
			if before.StartTime != result.StartTime || before.BookingCutoffTime != result.BookingCutoffTime || before.EndTime != result.EndTime {
				if serviceDate, relevant := currentWeekDateForWeekday(result.Weekday); relevant {
					if _, err := deleteBookingsForConfiguration(ctx, tx, operator.AccountID, BookingListFilter{
						ItemID: result.ItemID, ServiceDate: &serviceDate, Session: result.Session,
					}); err != nil {
						return nil, "", err
					}
				}
			}
			if err := tx.UpdateItemWindow(ctx, result, command.ExpectedVersion); err != nil {
				return nil, "", err
			}
		}
		return struct {
			Before *ItemWeeklyWindow
			After  ItemWeeklyWindow
		}{before, result}, result.WindowID, nil
	}, func(data []byte) error { return staffsupport.DecodeAfter(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}

// ListItemWindows 返回检查项目自己的每周预约窗口。
func (m *Manager) ListItemWindows(ctx context.Context, operator authn.Principal, itemID string, activeOnly bool) ([]ItemWeeklyWindow, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return nil, err
	}
	itemID, err := staffsupport.NormalizeUUID(itemID, "item_id")
	if err != nil {
		return nil, err
	}
	item, err := m.getItemSummary(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if err = staffsupport.RequireDepartmentScope(operator, item.DepartmentID); err != nil {
		return nil, err
	}
	key := fmt.Sprintf("department:%s:g:%s:item:%s:windows:%t", item.DepartmentID, m.generation(ctx, item.DepartmentID), itemID, activeOnly)
	result, _, err := staffsupport.LoadCached(ctx, m.cache, &m.flights, key, staffsupport.HotReadCacheTTL, func() ([]ItemWeeklyWindow, bool, error) {
		v, e := m.projects.ListItemWindows(ctx, itemID, activeOnly)
		return v, true, e
	})
	return result, err
}

// DisableItemWindow 停用项目预约窗口，并处理当前周受影响的预约。
func (m *Manager) DisableItemWindow(ctx context.Context, operator authn.Principal, command staffinput.ChangeStatus) (ItemWeeklyWindow, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ItemWeeklyWindow{}, err
	}
	id, err := staffsupport.NormalizeUUID(command.ResourceID, "window_id")
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	if command.ExpectedVersion < 1 {
		return ItemWeeklyWindow{}, ErrInvalid
	}
	command.ResourceID, command.Operation = id, meta
	var result ItemWeeklyWindow
	var departmentID string
	err = m.mutate(ctx, m.projects, operator, meta, "item_window", id, actionItemWindowDisabled, command, func() string { return departmentID }, func(tx ConfigurationTxStore) (any, string, error) {
		before, loadErr := tx.GetItemWindowForUpdate(ctx, id)
		if loadErr != nil {
			return nil, "", loadErr
		}
		item, loadErr := tx.GetItemForUpdate(ctx, before.ItemID)
		if loadErr != nil {
			return nil, "", loadErr
		}
		departmentID = item.DepartmentID
		if scopeErr := staffsupport.RequireDepartmentScope(operator, departmentID); scopeErr != nil {
			return nil, "", scopeErr
		}
		if before.Version != command.ExpectedVersion {
			return nil, "", ErrVersionConflict
		}
		after := before
		if after.Status != StatusDisabled {
			if serviceDate, relevant := currentWeekDateForWeekday(after.Weekday); relevant {
				if _, deleteErr := deleteBookingsForConfiguration(ctx, tx, operator.AccountID, BookingListFilter{
					ItemID: after.ItemID, ServiceDate: &serviceDate, Session: after.Session,
				}); deleteErr != nil {
					return nil, "", deleteErr
				}
			}
			after.Status = StatusDisabled
			after.Version++
			after.UpdatedAt = time.Now().UTC()
			if validationErr := staffsupport.ValidateItemWindowChange(ctx, tx, after); validationErr != nil {
				return nil, "", validationErr
			}
			if updateErr := tx.SetItemWindowStatus(ctx, after, command.ExpectedVersion); updateErr != nil {
				return nil, "", updateErr
			}
		}
		result = after
		return struct{ Before, After ItemWeeklyWindow }{before, after}, id, nil
	}, func(data []byte) error { return staffsupport.DecodeAfter(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}
