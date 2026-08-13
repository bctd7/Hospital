package staff

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

func (m *Manager) AddRoomItem(ctx context.Context, operator authn.Principal, command AddRoomItemCommand) (RoomItem, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return RoomItem{}, err
	}
	roomID, err := normalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return RoomItem{}, err
	}
	itemID, err := normalizeUUID(command.ItemID, "item_id")
	if err != nil {
		return RoomItem{}, err
	}
	meta, err := normalizeOperation(command.OperationMeta)
	if err != nil {
		return RoomItem{}, err
	}
	command.RoomID, command.ItemID, command.OperationMeta = roomID, itemID, meta
	var result RoomItem
	var departmentID string
	err = m.mutate(ctx, operator, meta, "room_item", "", actionRelationAdded, command, func() string { return departmentID }, func(tx RoomScheduleTxStore) (any, string, error) {
		room, lockErr := tx.GetRoomForUpdate(ctx, roomID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		item, lockErr := tx.GetItemForUpdate(ctx, itemID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = room.DepartmentID
		if err := requireScope(operator, departmentID); err != nil {
			return nil, "", err
		}
		if room.DepartmentID != item.DepartmentID {
			return nil, "", fmt.Errorf("%w: room and examination item must belong to the same department", ErrConflict)
		}
		if room.RetiredAt != nil || item.Status != StatusActive {
			return nil, "", ErrInvalidState
		}
		if existing, exists, findErr := tx.FindRelationForUpdate(ctx, roomID, itemID); findErr != nil {
			return nil, "", findErr
		} else if exists {
			return nil, "", fmt.Errorf("%w: room and item relation already exists as %s", ErrConflict, existing.Status)
		}
		if err := validatePair(ctx, tx, roomID, itemID, nil, nil); err != nil {
			return nil, "", err
		}
		now := time.Now().UTC()
		result = RoomItem{RelationID: uuid.NewString(), RoomID: roomID, ItemID: itemID, Status: StatusActive, Version: 1, CreatedAt: now, UpdatedAt: now}
		if err := tx.CreateRelation(ctx, result); err != nil {
			return nil, "", err
		}
		return result, result.RelationID, nil
	}, func(data []byte) error { return json.Unmarshal(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}

func (m *Manager) DisableRoomItem(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (RoomItem, error) {
	return m.changeRelationStatus(ctx, operator, command, StatusDisabled, actionRelationDisabled)
}
func (m *Manager) EnableRoomItem(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (RoomItem, error) {
	return m.changeRelationStatus(ctx, operator, command, StatusActive, actionRelationEnabled)
}

func (m *Manager) changeRelationStatus(ctx context.Context, operator authn.Principal, command ChangeStatusCommand, target Status, action string) (RoomItem, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return RoomItem{}, err
	}
	id, err := normalizeUUID(command.ResourceID, "relation_id")
	if err != nil {
		return RoomItem{}, err
	}
	meta, err := normalizeOperation(command.OperationMeta)
	if err != nil {
		return RoomItem{}, err
	}
	if command.ExpectedVersion < 1 {
		return RoomItem{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	command.ResourceID, command.OperationMeta = id, meta
	var result RoomItem
	var departmentID string
	err = m.mutate(ctx, operator, meta, "room_item", id, action, command, func() string { return departmentID }, func(tx RoomScheduleTxStore) (any, string, error) {
		before, lockErr := tx.GetRelationForUpdate(ctx, id)
		if lockErr != nil {
			return nil, "", lockErr
		}
		room, lockErr := tx.GetRoomForUpdate(ctx, before.RoomID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = room.DepartmentID
		if err := requireScope(operator, departmentID); err != nil {
			return nil, "", err
		}
		if before.Version != command.ExpectedVersion {
			return nil, "", ErrVersionConflict
		}
		if target == StatusActive {
			item, itemErr := tx.GetItemForUpdate(ctx, before.ItemID)
			if itemErr != nil {
				return nil, "", itemErr
			}
			if room.RetiredAt != nil || item.Status != StatusActive || room.DepartmentID != item.DepartmentID {
				return nil, "", ErrInvalidState
			}
			if err := validatePair(ctx, tx, before.RoomID, before.ItemID, nil, nil); err != nil {
				return nil, "", err
			}
		}
		result = before
		if result.Status != target {
			result.Status = target
			result.Version++
			result.UpdatedAt = time.Now().UTC()
			if err := tx.SetRelationStatus(ctx, result, command.ExpectedVersion); err != nil {
				return nil, "", err
			}
		}
		return struct{ Before, After RoomItem }{before, result}, id, nil
	}, func(data []byte) error { return decodeAfter(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}

func (m *Manager) ListRoomItems(ctx context.Context, operator authn.Principal, roomID string, query ListRelationsQuery) (Page[RoomItem], error) {
	room, err := m.GetRoom(ctx, operator, roomID)
	if err != nil {
		return Page[RoomItem]{}, err
	}
	if query.Status != "" && !query.Status.Valid() {
		return Page[RoomItem]{}, ErrInvalid
	}
	page, size, offset, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[RoomItem]{}, err
	}
	gen := m.generation(ctx, room.DepartmentID)
	key := fmt.Sprintf("department:%s:g:%s:room:%s:items:%s:%d:%d", room.DepartmentID, gen, room.RoomID, query.Status, page, size)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, hotReadCacheTTL, func() (Page[RoomItem], bool, error) {
		items, total, e := m.store.ListRoomItems(ctx, room.RoomID, query.Status, offset, size)
		return Page[RoomItem]{Items: items, Page: page, PageSize: size, Total: total}, true, e
	})
	return result, err
}
