package staff

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

const (
	actionRelationAdded    = "appointment.resource.room_item.added"
	actionRelationDisabled = "appointment.resource.room_item.disabled"
	actionRelationEnabled  = "appointment.resource.room_item.enabled"
)

// AddRoomItem 建立房间能够执行某个检查项目的关系。
func (m *Manager) AddRoomItem(ctx context.Context, operator authn.Principal, command staffinput.AddRoomItem) (RoomItem, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return RoomItem{}, err
	}
	roomID, err := staffsupport.NormalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return RoomItem{}, err
	}
	itemID, err := staffsupport.NormalizeUUID(command.ItemID, "item_id")
	if err != nil {
		return RoomItem{}, err
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return RoomItem{}, err
	}
	command.RoomID, command.ItemID, command.Operation = roomID, itemID, meta
	var result RoomItem
	var departmentID string
	err = m.mutate(ctx, m.rooms, operator, meta, "room_item", "", actionRelationAdded, command, func() string { return departmentID }, func(tx ConfigurationTxStore) (any, string, error) {
		room, lockErr := tx.GetRoomForUpdate(ctx, roomID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		item, lockErr := tx.GetItemForUpdate(ctx, itemID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = room.DepartmentID
		if err := staffsupport.RequireDepartmentScope(operator, departmentID); err != nil {
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
		if err := staffsupport.ValidatePair(ctx, tx, roomID, itemID, nil, nil); err != nil {
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

// DisableRoomItem 停用房间与检查项目的执行关系。
func (m *Manager) DisableRoomItem(ctx context.Context, operator authn.Principal, command staffinput.ChangeStatus) (RoomItem, error) {
	return m.changeRelationStatus(ctx, operator, command, StatusDisabled, actionRelationDisabled)
}

// EnableRoomItem 重新启用房间与检查项目的执行关系。
func (m *Manager) EnableRoomItem(ctx context.Context, operator authn.Principal, command staffinput.ChangeStatus) (RoomItem, error) {
	return m.changeRelationStatus(ctx, operator, command, StatusActive, actionRelationEnabled)
}

func (m *Manager) changeRelationStatus(ctx context.Context, operator authn.Principal, command staffinput.ChangeStatus, target Status, action string) (RoomItem, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return RoomItem{}, err
	}
	id, err := staffsupport.NormalizeUUID(command.ResourceID, "relation_id")
	if err != nil {
		return RoomItem{}, err
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return RoomItem{}, err
	}
	if command.ExpectedVersion < 1 {
		return RoomItem{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	command.ResourceID, command.Operation = id, meta
	var result RoomItem
	var departmentID string
	err = m.mutate(ctx, m.rooms, operator, meta, "room_item", id, action, command, func() string { return departmentID }, func(tx ConfigurationTxStore) (any, string, error) {
		before, lockErr := tx.GetRelationForUpdate(ctx, id)
		if lockErr != nil {
			return nil, "", lockErr
		}
		room, lockErr := tx.GetRoomForUpdate(ctx, before.RoomID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = room.DepartmentID
		if err := staffsupport.RequireDepartmentScope(operator, departmentID); err != nil {
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
			if err := staffsupport.ValidatePair(ctx, tx, before.RoomID, before.ItemID, nil, nil); err != nil {
				return nil, "", err
			}
		}
		result = before
		if result.Status != target {
			if target == StatusDisabled {
				today, weekEnd := currentBookingWeek()
				if _, err := deleteBookingsForConfiguration(ctx, tx, operator.AccountID, meta.OperationID, BookingListFilter{
					RoomID: before.RoomID, ItemID: before.ItemID, FromDate: &today, ThroughDate: &weekEnd,
				}); err != nil {
					return nil, "", err
				}
			}
			result.Status = target
			result.Version++
			result.UpdatedAt = time.Now().UTC()
			if err := tx.SetRelationStatus(ctx, result, command.ExpectedVersion); err != nil {
				return nil, "", err
			}
		}
		return struct{ Before, After RoomItem }{before, result}, id, nil
	}, func(data []byte) error { return staffsupport.DecodeAfter(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}

// ListRoomItems 分页返回某个房间可以执行的检查项目。
func (m *Manager) ListRoomItems(ctx context.Context, operator authn.Principal, roomID string, query staffinput.ListRelations) (Page[RoomItem], error) {
	room, err := m.GetRoom(ctx, operator, roomID)
	if err != nil {
		return Page[RoomItem]{}, err
	}
	if query.Status != "" && !query.Status.Valid() {
		return Page[RoomItem]{}, ErrInvalid
	}
	page, size, offset, err := staffsupport.NormalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[RoomItem]{}, err
	}
	gen := m.generation(ctx, room.DepartmentID)
	key := fmt.Sprintf("department:%s:g:%s:room:%s:items:%s:%d:%d", room.DepartmentID, gen, room.RoomID, query.Status, page, size)
	result, _, err := staffsupport.LoadCached(ctx, m.cache, &m.flights, key, staffsupport.HotReadCacheTTL, func() (Page[RoomItem], bool, error) {
		items, total, e := m.rooms.ListRoomItems(ctx, room.RoomID, query.Status, offset, size)
		return Page[RoomItem]{Items: items, Page: page, PageSize: size, Total: total}, true, e
	})
	return result, err
}
