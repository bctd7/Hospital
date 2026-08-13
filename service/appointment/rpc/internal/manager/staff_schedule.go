package manager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

func (m *StaffManager) SetRoomWindow(ctx context.Context, operator authn.Principal, command SetRoomWindowCommand) (RoomWeeklyWindow, error) {
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
	err = m.mutate(ctx, operator, meta, "room_window", windowID, actionRoomWindowSet, command, func() string { return departmentID }, func(tx RoomScheduleTxStore) (any, string, error) {
		room, lockErr := tx.GetRoomForUpdate(ctx, roomID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = room.DepartmentID
		if err := requireScope(operator, departmentID); err != nil {
			return nil, "", err
		}
		if room.RetiredAt != nil {
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

func (m *StaffManager) DisableRoomWindow(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (RoomWeeklyWindow, error) {
	value, err := m.disableWindow(ctx, operator, command, true)
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	return value.(RoomWeeklyWindow), nil
}

func (m *StaffManager) SetItemWindow(ctx context.Context, operator authn.Principal, command SetItemWindowCommand) (ItemWeeklyWindow, error) {
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
	err = m.mutate(ctx, operator, meta, "item_window", windowID, actionItemWindowSet, command, func() string { return departmentID }, func(tx RoomScheduleTxStore) (any, string, error) {
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

func (m *StaffManager) DisableItemWindow(ctx context.Context, operator authn.Principal, command ChangeStatusCommand) (ItemWeeklyWindow, error) {
	value, err := m.disableWindow(ctx, operator, command, false)
	if err != nil {
		return ItemWeeklyWindow{}, err
	}
	return value.(ItemWeeklyWindow), nil
}

func (m *StaffManager) disableWindow(ctx context.Context, operator authn.Principal, command ChangeStatusCommand, roomWindow bool) (any, error) {
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
	err = m.mutate(ctx, operator, meta, resourceType, id, action, command, func() string { return departmentID }, func(tx RoomScheduleTxStore) (any, string, error) {
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

func (m *StaffManager) ListRoomWindows(ctx context.Context, operator authn.Principal, roomID string, activeOnly bool) ([]RoomWeeklyWindow, error) {
	room, err := m.GetRoom(ctx, operator, roomID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("department:%s:g:%s:room:%s:windows:%t", room.DepartmentID, m.generation(ctx, room.DepartmentID), room.RoomID, activeOnly)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, hotReadCacheTTL, func() ([]RoomWeeklyWindow, bool, error) {
		v, e := m.store.ListRoomWindows(ctx, room.RoomID, activeOnly)
		return v, true, e
	})
	return result, err
}

func (m *StaffManager) ListItemWindows(ctx context.Context, operator authn.Principal, itemID string, activeOnly bool) ([]ItemWeeklyWindow, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
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
	if err = requireScope(operator, item.DepartmentID); err != nil {
		return nil, err
	}
	key := fmt.Sprintf("department:%s:g:%s:item:%s:windows:%t", item.DepartmentID, m.generation(ctx, item.DepartmentID), itemID, activeOnly)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, hotReadCacheTTL, func() ([]ItemWeeklyWindow, bool, error) {
		v, e := m.store.ListItemWindows(ctx, itemID, activeOnly)
		return v, true, e
	})
	return result, err
}
