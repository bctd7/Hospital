package manager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
)

const (
	actionRoomCreated        = "appointment.resource.room.created"
	actionRoomUpdated        = "appointment.resource.room.updated"
	actionRoomRetired        = "appointment.resource.room.retired"
	actionRelationAdded      = "appointment.resource.room_item.added"
	actionRelationDisabled   = "appointment.resource.room_item.disabled"
	actionRelationEnabled    = "appointment.resource.room_item.enabled"
	actionRoomWindowSet      = "appointment.resource.room_window.set"
	actionRoomWindowDisabled = "appointment.resource.room_window.disabled"
	actionItemWindowSet      = "appointment.resource.item_window.set"
	actionItemWindowDisabled = "appointment.resource.item_window.disabled"
)

func validatePair(ctx context.Context, tx RoomScheduleTxStore, roomID, itemID string, overrideRoom *RoomWeeklyWindow, overrideItem *ItemWeeklyWindow) error {
	rooms, err := tx.ListRoomWindowsForUpdate(ctx, roomID)
	if err != nil {
		return err
	}
	items, err := tx.ListItemWindowsForUpdate(ctx, itemID)
	if err != nil {
		return err
	}
	if overrideRoom != nil {
		rooms = replaceRoomWindow(rooms, *overrideRoom)
	}
	if overrideItem != nil {
		items = replaceItemWindow(items, *overrideItem)
	}
	for _, item := range items {
		if item.Status != StatusActive {
			continue
		}
		matched := false
		for _, room := range rooms {
			if room.Weekday == item.Weekday && room.Session == item.Session && containsWindow(room, item) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%w: room=%s item=%s weekday=%d session=%s", ErrWindowConflict, roomID, itemID, item.Weekday, item.Session)
		}
	}
	return nil
}

func validateRoomWindowChange(ctx context.Context, tx RoomScheduleTxStore, window RoomWeeklyWindow) error {
	relations, err := tx.ListRoomRelationsForUpdate(ctx, window.RoomID)
	if err != nil {
		return err
	}
	for _, relation := range relations {
		if relation.Status == StatusActive {
			if err := validatePair(ctx, tx, relation.RoomID, relation.ItemID, &window, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
func validateItemWindowChange(ctx context.Context, tx RoomScheduleTxStore, window ItemWeeklyWindow) error {
	relations, err := tx.ListItemRelationsForUpdate(ctx, window.ItemID)
	if err != nil {
		return err
	}
	for _, relation := range relations {
		if relation.Status == StatusActive {
			if err := validatePair(ctx, tx, relation.RoomID, relation.ItemID, nil, &window); err != nil {
				return err
			}
		}
	}
	return nil
}
func replaceRoomWindow(values []RoomWeeklyWindow, value RoomWeeklyWindow) []RoomWeeklyWindow {
	for i := range values {
		if values[i].WindowID == value.WindowID || (values[i].Weekday == value.Weekday && values[i].Session == value.Session) {
			values[i] = value
			return values
		}
	}
	return append(values, value)
}
func replaceItemWindow(values []ItemWeeklyWindow, value ItemWeeklyWindow) []ItemWeeklyWindow {
	for i := range values {
		if values[i].WindowID == value.WindowID || (values[i].Weekday == value.Weekday && values[i].Session == value.Session) {
			values[i] = value
			return values
		}
	}
	return append(values, value)
}

func (m *StaffManager) mutate(ctx context.Context, operator authn.Principal, meta OperationMeta, resourceType, resourceID, action string, payload any, departmentID func() string, apply func(RoomScheduleTxStore) (any, string, error), replay func([]byte) error) error {
	fingerprint := fingerprint(action, payload)
	return m.store.WithinRoomScheduleTransaction(ctx, func(tx RoomScheduleTxStore) error {
		operation, exists, err := tx.FindOperation(ctx, meta.OperationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operator.AccountID || operation.ResourceType != resourceType || operation.Action != action || operation.RequestFingerprint != fingerprint || (resourceID != "" && operation.ResourceID != resourceID) {
				return fmt.Errorf("%w: operation_id was already used", ErrConflict)
			}
			return replay(operation.ResultData)
		}
		result, actualID, err := apply(tx)
		if err != nil {
			return err
		}
		before, after := splitResult(result)
		return tx.RecordChange(ctx, RoomScheduleChange{OperationID: meta.OperationID, OperatorAccountID: operator.AccountID, DepartmentID: departmentID(), ResourceType: resourceType, ResourceID: actualID, Action: action, RequestFingerprint: fingerprint, Before: before, After: after, RequestID: meta.RequestID})
	})
}

func splitResult(value any) (any, any) {
	data, _ := json.Marshal(value)
	var envelope map[string]json.RawMessage
	if json.Unmarshal(data, &envelope) == nil {
		if after, ok := envelope["After"]; ok {
			var a any
			_ = json.Unmarshal(after, &a)
			var b any
			if before, exists := envelope["Before"]; exists && string(before) != "null" {
				_ = json.Unmarshal(before, &b)
			}
			return b, a
		}
	}
	return nil, value
}
func departmentFromResult(value any) string {
	data, _ := json.Marshal(value)
	var direct struct {
		DepartmentID string `json:"department_id"`
		After        struct {
			DepartmentID string `json:"department_id"`
		} `json:"After"`
	}
	_ = json.Unmarshal(data, &direct)
	if direct.DepartmentID != "" {
		return direct.DepartmentID
	}
	return direct.After.DepartmentID
}
func decodeAfter(data []byte, target any) error {
	var envelope struct{ After json.RawMessage }
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	if len(envelope.After) == 0 {
		return json.Unmarshal(data, target)
	}
	return json.Unmarshal(envelope.After, target)
}
func fingerprint(action string, payload any) string {
	data, _ := json.Marshal(struct {
		Action  string
		Payload any
	}{action, payload})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func requirePermission(operator authn.Principal, permission string) error {
	if err := commonauthz.RequirePermission(operator, permission); err != nil {
		return fmt.Errorf("%w: %v", ErrForbidden, err)
	}
	if _, err := uuid.Parse(strings.TrimSpace(operator.AccountID)); err != nil {
		return fmt.Errorf("%w: invalid operator identity", ErrForbidden)
	}
	return nil
}

func requireScope(operator authn.Principal, departmentID string) error {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return nil
	}
	if !operator.HasRole(authn.RoleDepartmentDoctor) {
		return ErrForbidden
	}
	own, err := normalizeUUID(operator.DepartmentID, "operator department_id")
	if err != nil || own != departmentID {
		return ErrForbidden
	}
	return nil
}
func scopedDepartment(operator authn.Principal, requested string) (string, error) {
	if operator.HasRole(authn.RoleSuperAdmin) {
		return normalizeUUID(requested, "department_id")
	}
	own, err := normalizeUUID(operator.DepartmentID, "operator department_id")
	if err != nil {
		return "", ErrForbidden
	}
	if requested != "" {
		requested, err = normalizeUUID(requested, "department_id")
		if err != nil || requested != own {
			return "", ErrForbidden
		}
	}
	return own, nil
}
func (m *StaffManager) generation(ctx context.Context, departmentID string) string {
	if m.cache == nil {
		return "0"
	}
	value, err := m.cache.DepartmentGeneration(ctx, departmentID)
	if err != nil {
		return "0"
	}
	return value
}

func (m *StaffManager) getItemSummary(ctx context.Context, itemID string) (ItemSummary, error) {
	value, found, err := loadCached(ctx, m.cache, &m.flights, "item-summary:"+itemID, hotReadCacheTTL, func() (ItemSummary, bool, error) {
		item, loadErr := m.store.GetItemSummary(ctx, itemID)
		if errors.Is(loadErr, ErrNotFound) {
			return ItemSummary{}, false, nil
		}
		return item, loadErr == nil, loadErr
	})
	if err != nil {
		return ItemSummary{}, err
	}
	if !found {
		return ItemSummary{}, ErrNotFound
	}
	return value, nil
}

// InvalidateItem is called after a project transaction commits so patient-facing
// item summaries and all department-scoped projections switch generations.
func (m *StaffManager) InvalidateItem(ctx context.Context, departmentID, itemID string) {
	m.invalidate(ctx, departmentID, "item-summary:"+itemID)
}
func (m *StaffManager) invalidate(ctx context.Context, departmentID string, keys ...string) {
	if m.cache == nil {
		return
	}
	if departmentID != "" {
		_ = m.cache.BumpDepartment(ctx, departmentID)
	}
	_ = m.cache.Delete(ctx, keys...)
}
