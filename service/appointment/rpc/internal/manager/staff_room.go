package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

func (m *StaffManager) CreateRoom(ctx context.Context, operator authn.Principal, command CreateRoomCommand) (Room, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentCreate); err != nil {
		return Room{}, err
	}
	departmentID, err := normalizeUUID(command.DepartmentID, "department_id")
	if err != nil {
		return Room{}, err
	}
	campusID, building, floorNumber, roomNumber, displayName, err := normalizeRoomLocation(command.CampusID, command.Building, command.FloorNumber, command.RoomNumber)
	if err != nil {
		return Room{}, err
	}
	meta, err := normalizeOperation(command.OperationMeta)
	if err != nil {
		return Room{}, err
	}
	if err = requireScope(operator, departmentID); err != nil {
		return Room{}, err
	}
	command.DepartmentID, command.CampusID, command.Building, command.FloorNumber, command.RoomNumber, command.OperationMeta = departmentID, campusID, building, floorNumber, roomNumber, meta
	var result Room
	err = m.mutate(ctx, operator, meta, "room", "", actionRoomCreated, command, func() string { return departmentID }, func(tx RoomScheduleTxStore) (any, string, error) {
		now := time.Now().UTC()
		result = Room{RoomID: uuid.NewString(), DepartmentID: departmentID, CampusID: campusID, Building: building, FloorNumber: floorNumber, RoomNumber: roomNumber, DisplayName: displayName, Version: 1, CreatedAt: now, UpdatedAt: now}
		if err := tx.CreateRoom(ctx, result); err != nil {
			return nil, "", err
		}
		return result, result.RoomID, nil
	}, func(data []byte) error { return json.Unmarshal(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID, "room:"+result.RoomID)
	}
	return result, err
}

func (m *StaffManager) GetRoom(ctx context.Context, operator authn.Principal, roomID string) (Room, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Room{}, err
	}
	roomID, err := normalizeUUID(roomID, "room_id")
	if err != nil {
		return Room{}, err
	}
	room, found, err := loadCached(ctx, m.cache, &m.flights, "room:"+roomID, hotReadCacheTTL, func() (Room, bool, error) {
		value, loadErr := m.store.GetRoom(ctx, roomID)
		if errors.Is(loadErr, ErrNotFound) {
			return Room{}, false, nil
		}
		return value, loadErr == nil, loadErr
	})
	if err != nil {
		return Room{}, err
	}
	if !found {
		return Room{}, ErrNotFound
	}
	if err := requireScope(operator, room.DepartmentID); err != nil {
		return Room{}, err
	}
	return room, nil
}

func (m *StaffManager) ListRooms(ctx context.Context, operator authn.Principal, query ListRoomsQuery) (Page[Room], error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Page[Room]{}, err
	}
	departmentID, err := scopedDepartment(operator, query.DepartmentID)
	if err != nil {
		return Page[Room]{}, err
	}
	page, size, offset, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[Room]{}, err
	}
	gen := "0"
	if m.cache != nil {
		if value, genErr := m.cache.DepartmentGeneration(ctx, departmentID); genErr == nil {
			gen = value
		}
	}
	key := fmt.Sprintf("department:%s:g:%s:rooms:%d:%d", departmentID, gen, page, size)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, hotReadCacheTTL, func() (Page[Room], bool, error) {
		items, total, loadErr := m.store.ListRooms(ctx, departmentID, offset, size)
		return Page[Room]{Items: items, Page: page, PageSize: size, Total: total}, true, loadErr
	})
	return result, err
}

func (m *StaffManager) UpdateRoom(ctx context.Context, operator authn.Principal, command UpdateRoomCommand) (Room, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return Room{}, err
	}
	roomID, err := normalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return Room{}, err
	}
	campusID, building, floorNumber, roomNumber, displayName, err := normalizeRoomLocation(command.CampusID, command.Building, command.FloorNumber, command.RoomNumber)
	if err != nil {
		return Room{}, err
	}
	meta, err := normalizeOperation(command.OperationMeta)
	if err != nil {
		return Room{}, err
	}
	if command.ExpectedVersion < 1 {
		return Room{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	command.RoomID, command.CampusID, command.Building, command.FloorNumber, command.RoomNumber, command.OperationMeta = roomID, campusID, building, floorNumber, roomNumber, meta
	var result Room
	var departmentID string
	err = m.mutate(ctx, operator, meta, "room", roomID, actionRoomUpdated, command, func() string { return departmentID }, func(tx RoomScheduleTxStore) (any, string, error) {
		before, lockErr := tx.GetRoomForUpdate(ctx, roomID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = before.DepartmentID
		if err := requireScope(operator, departmentID); err != nil {
			return nil, "", err
		}
		if before.Version != command.ExpectedVersion {
			return nil, "", ErrVersionConflict
		}
		if before.RetiredAt != nil {
			return nil, "", ErrInvalidState
		}
		result = before
		result.CampusID = campusID
		result.Building = building
		result.FloorNumber = floorNumber
		result.RoomNumber = roomNumber
		result.DisplayName = displayName
		result.Version++
		result.UpdatedAt = time.Now().UTC()
		if err := tx.UpdateRoom(ctx, result, command.ExpectedVersion); err != nil {
			return nil, "", err
		}
		return struct{ Before, After Room }{before, result}, result.RoomID, nil
	}, func(data []byte) error { return decodeAfter(data, &result) })
	if departmentID == "" {
		departmentID = result.DepartmentID
	}
	if err == nil {
		m.invalidate(ctx, departmentID, "room:"+roomID)
	}
	return result, err
}

func (m *StaffManager) RetireRoom(ctx context.Context, operator authn.Principal, command RetireRoomCommand) (Room, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return Room{}, err
	}
	id, err := normalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return Room{}, err
	}
	meta, err := normalizeOperation(command.OperationMeta)
	if err != nil {
		return Room{}, err
	}
	if command.ExpectedVersion < 1 {
		return Room{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	command.RoomID, command.OperationMeta = id, meta
	var result Room
	var departmentID string
	err = m.mutate(ctx, operator, meta, "room", id, actionRoomRetired, command, func() string { return departmentID }, func(tx RoomScheduleTxStore) (any, string, error) {
		before, lockErr := tx.GetRoomForUpdate(ctx, id)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = before.DepartmentID
		if err := requireScope(operator, departmentID); err != nil {
			return nil, "", err
		}
		if before.Version != command.ExpectedVersion {
			return nil, "", ErrVersionConflict
		}
		if before.RetiredAt != nil {
			return nil, "", ErrInvalidState
		}
		result = before
		now := time.Now().UTC()
		result.RetiredAt = &now
		result.Version++
		result.UpdatedAt = now
		if err := tx.RetireRoom(ctx, result, command.ExpectedVersion); err != nil {
			return nil, "", err
		}
		return struct{ Before, After Room }{before, result}, id, nil
	}, func(data []byte) error { return decodeAfter(data, &result) })
	if departmentID == "" {
		departmentID = result.DepartmentID
	}
	if err == nil {
		m.invalidate(ctx, departmentID, "room:"+id)
	}
	return result, err
}
