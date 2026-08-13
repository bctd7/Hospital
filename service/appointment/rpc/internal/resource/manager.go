package resource

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
	actionRoomCreated        = "appointment.resource.room.created"
	actionRoomUpdated        = "appointment.resource.room.updated"
	actionRoomDisabled       = "appointment.resource.room.disabled"
	actionRoomEnabled        = "appointment.resource.room.enabled"
	actionRelationAdded      = "appointment.resource.room_item.added"
	actionRelationDisabled   = "appointment.resource.room_item.disabled"
	actionRelationEnabled    = "appointment.resource.room_item.enabled"
	actionRoomWindowSet      = "appointment.resource.room_window.set"
	actionRoomWindowDisabled = "appointment.resource.room_window.disabled"
	actionItemWindowSet      = "appointment.resource.item_window.set"
	actionItemWindowDisabled = "appointment.resource.item_window.disabled"
)

type Manager struct {
	store   Store
	cache   Cache
	flights flightGroup
}

func NewManager(store Store, cache Cache) (*Manager, error) {
	if store == nil {
		return nil, errors.New("appointment resource store is required")
	}
	return &Manager{store: store, cache: cache}, nil
}

func (m *Manager) CreateRoom(ctx context.Context, operator authn.Principal, command CreateRoomCommand) (Room, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentCreate); err != nil {
		return Room{}, err
	}
	departmentID, err := normalizeUUID(command.DepartmentID, "department_id")
	if err != nil {
		return Room{}, err
	}
	name, err := normalizeName(command.Name)
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
	command.DepartmentID, command.Name, command.OperationMeta = departmentID, name, meta
	var result Room
	err = m.mutate(ctx, operator, meta, "room", "", actionRoomCreated, command, func() string { return departmentID }, func(tx TxStore) (any, string, error) {
		now := time.Now().UTC()
		result = Room{RoomID: uuid.NewString(), DepartmentID: departmentID, Name: name, Status: StatusActive, Version: 1, CreatedAt: now, UpdatedAt: now}
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

func (m *Manager) GetRoom(ctx context.Context, operator authn.Principal, roomID string) (Room, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Room{}, err
	}
	roomID, err := normalizeUUID(roomID, "room_id")
	if err != nil {
		return Room{}, err
	}
	room, found, err := loadCached(ctx, m.cache, &m.flights, "room:"+roomID, resourceCacheTTL, func() (Room, bool, error) {
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

func (m *Manager) ListRooms(ctx context.Context, operator authn.Principal, query ListRoomsQuery) (Page[Room], error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Page[Room]{}, err
	}
	departmentID, err := scopedDepartment(operator, query.DepartmentID)
	if err != nil {
		return Page[Room]{}, err
	}
	if query.Status != "" && !query.Status.Valid() {
		return Page[Room]{}, fmt.Errorf("%w: invalid status", ErrInvalid)
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
	key := fmt.Sprintf("department:%s:g:%s:rooms:%s:%d:%d", departmentID, gen, query.Status, page, size)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, resourceCacheTTL, func() (Page[Room], bool, error) {
		items, total, loadErr := m.store.ListRooms(ctx, departmentID, query.Status, offset, size)
		return Page[Room]{Items: items, Page: page, PageSize: size, Total: total}, true, loadErr
	})
	return result, err
}

func (m *Manager) UpdateRoom(ctx context.Context, operator authn.Principal, command UpdateRoomCommand) (Room, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return Room{}, err
	}
	roomID, err := normalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return Room{}, err
	}
	name, err := normalizeName(command.Name)
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
	command.RoomID, command.Name, command.OperationMeta = roomID, name, meta
	var result Room
	var departmentID string
	err = m.mutate(ctx, operator, meta, "room", roomID, actionRoomUpdated, command, func() string { return departmentID }, func(tx TxStore) (any, string, error) {
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
		if before.Status != StatusActive {
			return nil, "", ErrInvalidState
		}
		result = before
		result.Name = name
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

func (m *Manager) DisableRoom(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (Room, error) {
	return m.changeRoomStatus(ctx, operator, command, StatusDisabled, actionRoomDisabled)
}
func (m *Manager) EnableRoom(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (Room, error) {
	return m.changeRoomStatus(ctx, operator, command, StatusActive, actionRoomEnabled)
}

func (m *Manager) changeRoomStatus(ctx context.Context, operator authn.Principal, command ChangeStatusCommand, target Status, action string) (Room, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return Room{}, err
	}
	id, err := normalizeUUID(command.ResourceID, "room_id")
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
	command.ResourceID, command.OperationMeta = id, meta
	var result Room
	var departmentID string
	err = m.mutate(ctx, operator, meta, "room", id, action, command, func() string { return departmentID }, func(tx TxStore) (any, string, error) {
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
		result = before
		if result.Status != target {
			result.Status = target
			result.Version++
			result.UpdatedAt = time.Now().UTC()
			if err := tx.SetRoomStatus(ctx, result, command.ExpectedVersion); err != nil {
				return nil, "", err
			}
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
	err = m.mutate(ctx, operator, meta, "room_item", "", actionRelationAdded, command, func() string { return departmentID }, func(tx TxStore) (any, string, error) {
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
		if room.Status != StatusActive || item.Status != StatusActive {
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
	err = m.mutate(ctx, operator, meta, "room_item", id, action, command, func() string { return departmentID }, func(tx TxStore) (any, string, error) {
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
			if room.Status != StatusActive || item.Status != StatusActive || room.DepartmentID != item.DepartmentID {
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
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, resourceCacheTTL, func() (Page[RoomItem], bool, error) {
		items, total, e := m.store.ListRoomItems(ctx, room.RoomID, query.Status, offset, size)
		return Page[RoomItem]{Items: items, Page: page, PageSize: size, Total: total}, true, e
	})
	return result, err
}

func (m *Manager) ListItemRooms(ctx context.Context, operator authn.Principal, itemID string, activeOnly bool) ([]RoomItem, error) {
	if err := requirePatientReadOrPermission(operator); err != nil {
		return nil, err
	}
	itemID, err := normalizeUUID(itemID, "item_id")
	if err != nil {
		return nil, err
	}
	item, err := m.getItemSummary(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if operator.AccountType == authn.AccountTypePatient {
		activeOnly = true
		if item.Status != StatusActive {
			return nil, ErrNotFound
		}
	} else if err := requireScope(operator, item.DepartmentID); err != nil {
		return nil, err
	}
	key := fmt.Sprintf("department:%s:g:%s:item:%s:rooms:%t", item.DepartmentID, m.generation(ctx, item.DepartmentID), itemID, activeOnly)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, queryCacheTTL, func() ([]RoomItem, bool, error) {
		value, e := m.store.ListItemRooms(ctx, itemID, activeOnly)
		return value, true, e
	})
	return result, err
}

func (m *Manager) SetRoomWindow(ctx context.Context, operator authn.Principal, command SetRoomWindowCommand) (RoomWeeklyWindow, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return RoomWeeklyWindow{}, err
	}
	roomID, err := normalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	windowID := ""
	if strings.TrimSpace(command.WindowID) != "" {
		windowID, err = normalizeUUID(command.WindowID, "window_id")
		if err != nil {
			return RoomWeeklyWindow{}, err
		}
	}
	weekday, session, err := normalizeSlot(command.Weekday, command.Session)
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	openTime, openMinutes, err := normalizeClock(command.OpenTime, "open_time")
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	closeTime, closeMinutes, err := normalizeClock(command.CloseTime, "close_time")
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	if openMinutes >= closeMinutes || command.ActiveCapacity < 1 {
		return RoomWeeklyWindow{}, fmt.Errorf("%w: room window time or capacity is invalid", ErrInvalid)
	}
	meta, err := normalizeOperation(command.OperationMeta)
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	if windowID == "" && command.ExpectedVersion != 0 {
		return RoomWeeklyWindow{}, fmt.Errorf("%w: expected_version must be zero for a new window", ErrInvalid)
	}
	if windowID != "" && command.ExpectedVersion < 1 {
		return RoomWeeklyWindow{}, fmt.Errorf("%w: expected_version is required", ErrInvalid)
	}
	command.WindowID, command.RoomID, command.Weekday, command.Session, command.OpenTime, command.CloseTime, command.OperationMeta = windowID, roomID, weekday, session, openTime, closeTime, meta
	var result RoomWeeklyWindow
	var departmentID string
	err = m.mutate(ctx, operator, meta, "room_window", windowID, actionRoomWindowSet, command, func() string { return departmentID }, func(tx TxStore) (any, string, error) {
		room, lockErr := tx.GetRoomForUpdate(ctx, roomID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = room.DepartmentID
		if err := requireScope(operator, departmentID); err != nil {
			return nil, "", err
		}
		if room.Status != StatusActive {
			return nil, "", ErrInvalidState
		}
		now := time.Now().UTC()
		var before *RoomWeeklyWindow
		if windowID == "" {
			if _, exists, e := tx.FindRoomWindowForUpdate(ctx, roomID, weekday, session); e != nil {
				return nil, "", e
			} else if exists {
				return nil, "", fmt.Errorf("%w: room window slot already exists", ErrConflict)
			}
			result = RoomWeeklyWindow{WindowID: uuid.NewString(), RoomID: roomID, Weekday: weekday, Session: session, OpenTime: openTime, CloseTime: closeTime, ActiveCapacity: command.ActiveCapacity, Status: StatusActive, Version: 1, CreatedAt: now, UpdatedAt: now}
		} else {
			current, e := tx.GetRoomWindowForUpdate(ctx, windowID)
			if e != nil {
				return nil, "", e
			}
			if current.RoomID != roomID || current.Weekday != weekday || current.Session != session {
				return nil, "", ErrConflict
			}
			if current.Version != command.ExpectedVersion {
				return nil, "", ErrVersionConflict
			}
			before = &current
			result = current
			result.OpenTime, result.CloseTime, result.ActiveCapacity, result.Status, result.Version, result.UpdatedAt = openTime, closeTime, command.ActiveCapacity, StatusActive, current.Version+1, now
		}
		if err := validateRoomWindowChange(ctx, tx, result); err != nil {
			return nil, "", err
		}
		if before == nil {
			if err := tx.CreateRoomWindow(ctx, result); err != nil {
				return nil, "", err
			}
		} else if err := tx.UpdateRoomWindow(ctx, result, command.ExpectedVersion); err != nil {
			return nil, "", err
		}
		return struct {
			Before *RoomWeeklyWindow
			After  RoomWeeklyWindow
		}{before, result}, result.WindowID, nil
	}, func(data []byte) error { return decodeAfter(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}

func (m *Manager) DisableRoomWindow(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (RoomWeeklyWindow, error) {
	value, err := m.disableWindow(ctx, operator, command, true)
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	return value.(RoomWeeklyWindow), nil
}

func (m *Manager) SetItemWindow(ctx context.Context, operator authn.Principal, command SetItemWindowCommand) (ItemWeeklyWindow, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ItemWeeklyWindow{}, err
	}
	itemID, err := normalizeUUID(command.ItemID, "item_id")
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	windowID := ""
	if strings.TrimSpace(command.WindowID) != "" {
		windowID, err = normalizeUUID(command.WindowID, "window_id")
		if err != nil {
			return ItemWeeklyWindow{}, err
		}
	}
	weekday, session, err := normalizeSlot(command.Weekday, command.Session)
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	start, startMin, err := normalizeClock(command.StartTime, "start_time")
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	cutoff, cutoffMin, err := normalizeClock(command.BookingCutoffTime, "booking_cutoff_time")
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	end, endMin, err := normalizeClock(command.EndTime, "end_time")
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	if startMin > cutoffMin || cutoffMin >= endMin {
		return ItemWeeklyWindow{}, fmt.Errorf("%w: item window time is invalid", ErrInvalid)
	}
	meta, err := normalizeOperation(command.OperationMeta)
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	if windowID == "" && command.ExpectedVersion != 0 {
		return ItemWeeklyWindow{}, fmt.Errorf("%w: expected_version must be zero for a new window", ErrInvalid)
	}
	if windowID != "" && command.ExpectedVersion < 1 {
		return ItemWeeklyWindow{}, fmt.Errorf("%w: expected_version is required", ErrInvalid)
	}
	command.WindowID, command.ItemID, command.Weekday, command.Session, command.StartTime, command.BookingCutoffTime, command.EndTime, command.OperationMeta = windowID, itemID, weekday, session, start, cutoff, end, meta
	var result ItemWeeklyWindow
	var departmentID string
	err = m.mutate(ctx, operator, meta, "item_window", windowID, actionItemWindowSet, command, func() string { return departmentID }, func(tx TxStore) (any, string, error) {
		item, lockErr := tx.GetItemForUpdate(ctx, itemID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = item.DepartmentID
		if err := requireScope(operator, departmentID); err != nil {
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
		if err := validateItemWindowChange(ctx, tx, result); err != nil {
			return nil, "", err
		}
		if before == nil {
			if err := tx.CreateItemWindow(ctx, result); err != nil {
				return nil, "", err
			}
		} else if err := tx.UpdateItemWindow(ctx, result, command.ExpectedVersion); err != nil {
			return nil, "", err
		}
		return struct {
			Before *ItemWeeklyWindow
			After  ItemWeeklyWindow
		}{before, result}, result.WindowID, nil
	}, func(data []byte) error { return decodeAfter(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}

func (m *Manager) DisableItemWindow(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (ItemWeeklyWindow, error) {
	value, err := m.disableWindow(ctx, operator, command, false)
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	return value.(ItemWeeklyWindow), nil
}

func (m *Manager) disableWindow(ctx context.Context, operator authn.Principal, command ChangeStatusCommand, roomWindow bool) (any, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return nil, err
	}
	id, err := normalizeUUID(command.ResourceID, "window_id")
	if err != nil {
		return nil, err
	}
	meta, err := normalizeOperation(command.OperationMeta)
	if err != nil {
		return nil, err
	}
	if command.ExpectedVersion < 1 {
		return nil, ErrInvalid
	}
	command.ResourceID, command.OperationMeta = id, meta
	resourceType, action := "item_window", actionItemWindowDisabled
	if roomWindow {
		resourceType, action = "room_window", actionRoomWindowDisabled
	}
	var result any
	var departmentID string
	err = m.mutate(ctx, operator, meta, resourceType, id, action, command, func() string { return departmentID }, func(tx TxStore) (any, string, error) {
		if roomWindow {
			before, e := tx.GetRoomWindowForUpdate(ctx, id)
			if e != nil {
				return nil, "", e
			}
			room, e := tx.GetRoomForUpdate(ctx, before.RoomID)
			if e != nil {
				return nil, "", e
			}
			departmentID = room.DepartmentID
			if e = requireScope(operator, departmentID); e != nil {
				return nil, "", e
			}
			if before.Version != command.ExpectedVersion {
				return nil, "", ErrVersionConflict
			}
			after := before
			if after.Status != StatusDisabled {
				after.Status = StatusDisabled
				after.Version++
				after.UpdatedAt = time.Now().UTC()
				if e = validateRoomWindowChange(ctx, tx, after); e != nil {
					return nil, "", e
				}
				if e = tx.SetRoomWindowStatus(ctx, after, command.ExpectedVersion); e != nil {
					return nil, "", e
				}
			}
			result = after
			return struct{ Before, After RoomWeeklyWindow }{before, after}, id, nil
		}
		before, e := tx.GetItemWindowForUpdate(ctx, id)
		if e != nil {
			return nil, "", e
		}
		item, e := tx.GetItemForUpdate(ctx, before.ItemID)
		if e != nil {
			return nil, "", e
		}
		departmentID = item.DepartmentID
		if e = requireScope(operator, departmentID); e != nil {
			return nil, "", e
		}
		if before.Version != command.ExpectedVersion {
			return nil, "", ErrVersionConflict
		}
		after := before
		if after.Status != StatusDisabled {
			after.Status = StatusDisabled
			after.Version++
			after.UpdatedAt = time.Now().UTC()
			if e = validateItemWindowChange(ctx, tx, after); e != nil {
				return nil, "", e
			}
			if e = tx.SetItemWindowStatus(ctx, after, command.ExpectedVersion); e != nil {
				return nil, "", e
			}
		}
		result = after
		return struct{ Before, After ItemWeeklyWindow }{before, after}, id, nil
	}, func(data []byte) error {
		if roomWindow {
			var value RoomWeeklyWindow
			if e := decodeAfter(data, &value); e != nil {
				return e
			}
			result = value
		} else {
			var value ItemWeeklyWindow
			if e := decodeAfter(data, &value); e != nil {
				return e
			}
			result = value
		}
		return nil
	})
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}

func (m *Manager) ListRoomWindows(ctx context.Context, operator authn.Principal, roomID string, activeOnly bool) ([]RoomWeeklyWindow, error) {
	room, err := m.GetRoom(ctx, operator, roomID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("department:%s:g:%s:room:%s:windows:%t", room.DepartmentID, m.generation(ctx, room.DepartmentID), room.RoomID, activeOnly)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, resourceCacheTTL, func() ([]RoomWeeklyWindow, bool, error) {
		v, e := m.store.ListRoomWindows(ctx, room.RoomID, activeOnly)
		return v, true, e
	})
	return result, err
}

func (m *Manager) ListItemWindows(ctx context.Context, operator authn.Principal, itemID string, activeOnly bool) ([]ItemWeeklyWindow, error) {
	if err := requirePatientReadOrPermission(operator); err != nil {
		return nil, err
	}
	itemID, err := normalizeUUID(itemID, "item_id")
	if err != nil {
		return nil, err
	}
	item, err := m.getItemSummary(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if operator.AccountType == authn.AccountTypePatient {
		activeOnly = true
		if item.Status != StatusActive {
			return nil, ErrNotFound
		}
	} else if err = requireScope(operator, item.DepartmentID); err != nil {
		return nil, err
	}
	key := fmt.Sprintf("department:%s:g:%s:item:%s:windows:%t", item.DepartmentID, m.generation(ctx, item.DepartmentID), itemID, activeOnly)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, resourceCacheTTL, func() ([]ItemWeeklyWindow, bool, error) {
		v, e := m.store.ListItemWindows(ctx, itemID, activeOnly)
		return v, true, e
	})
	return result, err
}

func validatePair(ctx context.Context, tx TxStore, roomID, itemID string, overrideRoom *RoomWeeklyWindow, overrideItem *ItemWeeklyWindow) error {
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

func validateRoomWindowChange(ctx context.Context, tx TxStore, window RoomWeeklyWindow) error {
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
func validateItemWindowChange(ctx context.Context, tx TxStore, window ItemWeeklyWindow) error {
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

func (m *Manager) mutate(ctx context.Context, operator authn.Principal, meta OperationMeta, resourceType, resourceID, action string, payload any, departmentID func() string, apply func(TxStore) (any, string, error), replay func([]byte) error) error {
	fingerprint := fingerprint(action, payload)
	return m.store.WithinResourceTransaction(ctx, func(tx TxStore) error {
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
		return tx.RecordChange(ctx, Change{OperationID: meta.OperationID, OperatorAccountID: operator.AccountID, DepartmentID: departmentID(), ResourceType: resourceType, ResourceID: actualID, Action: action, RequestFingerprint: fingerprint, Before: before, After: after, RequestID: meta.RequestID})
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

func requirePatientReadOrPermission(operator authn.Principal) error {
	if operator.AccountType != authn.AccountTypePatient {
		return requirePermission(operator, contractauthz.PermissionAppointmentRead)
	}
	if _, err := uuid.Parse(strings.TrimSpace(operator.AccountID)); err != nil {
		return fmt.Errorf("%w: invalid patient identity", ErrForbidden)
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
func (m *Manager) generation(ctx context.Context, departmentID string) string {
	if m.cache == nil {
		return "0"
	}
	value, err := m.cache.DepartmentGeneration(ctx, departmentID)
	if err != nil {
		return "0"
	}
	return value
}

func (m *Manager) getItemSummary(ctx context.Context, itemID string) (ItemSummary, error) {
	value, found, err := loadCached(ctx, m.cache, &m.flights, "item-summary:"+itemID, resourceCacheTTL, func() (ItemSummary, bool, error) {
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

// InvalidateItem is called after a catalog transaction commits so patient-facing
// item summaries and all department-scoped projections switch generations.
func (m *Manager) InvalidateItem(ctx context.Context, departmentID, itemID string) {
	m.invalidate(ctx, departmentID, "item-summary:"+itemID)
}
func (m *Manager) invalidate(ctx context.Context, departmentID string, keys ...string) {
	if m.cache == nil {
		return
	}
	if departmentID != "" {
		_ = m.cache.BumpDepartment(ctx, departmentID)
	}
	_ = m.cache.Delete(ctx, keys...)
}
