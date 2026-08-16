package staff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

const (
	actionRoomCreated        = "appointment.resource.room.created"
	actionRoomUpdated        = "appointment.resource.room.updated"
	actionRoomRetired        = "appointment.resource.room.retired"
	actionRoomWindowSet      = "appointment.resource.room_window.set"
	actionRoomWindowDisabled = "appointment.resource.room_window.disabled"
)

// CreateRoom 创建隶属于指定科室、具有严格地址字段的房间。
func (m *Manager) CreateRoom(ctx context.Context, operator authn.Principal, command staffinput.CreateRoom) (Room, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentCreate); err != nil {
		return Room{}, err
	}
	departmentID, err := staffsupport.NormalizeUUID(command.DepartmentID, "department_id")
	if err != nil {
		return Room{}, err
	}
	campusID, building, floorNumber, roomNumber, displayName, err := staffsupport.NormalizeRoomLocation(command.CampusID, command.Building, command.FloorNumber, command.RoomNumber)
	if err != nil {
		return Room{}, err
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return Room{}, err
	}
	if err = staffsupport.RequireDepartmentScope(operator, departmentID); err != nil {
		return Room{}, err
	}
	command.DepartmentID, command.CampusID, command.Building, command.FloorNumber, command.RoomNumber, command.Operation = departmentID, campusID, building, floorNumber, roomNumber, meta
	var result Room
	err = m.mutate(ctx, m.rooms, operator, meta, "room", "", actionRoomCreated, command, func() string { return departmentID }, func(tx ConfigurationTxStore) (any, string, error) {
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

// GetRoom 返回工作人员权限范围内的房间详情。
func (m *Manager) GetRoom(ctx context.Context, operator authn.Principal, roomID string) (Room, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Room{}, err
	}
	roomID, err := staffsupport.NormalizeUUID(roomID, "room_id")
	if err != nil {
		return Room{}, err
	}
	room, found, err := staffsupport.LoadCached(ctx, m.cache, &m.flights, "room:"+roomID, staffsupport.HotReadCacheTTL, func() (Room, bool, error) {
		value, loadErr := m.rooms.GetRoom(ctx, roomID)
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
	if err := staffsupport.RequireDepartmentScope(operator, room.DepartmentID); err != nil {
		return Room{}, err
	}
	return room, nil
}

// ListRooms 分页返回科室所属房间。
func (m *Manager) ListRooms(ctx context.Context, operator authn.Principal, query staffinput.ListRooms) (Page[Room], error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Page[Room]{}, err
	}
	departmentID, err := staffsupport.ScopedDepartment(operator, query.DepartmentID)
	if err != nil {
		return Page[Room]{}, err
	}
	page, size, offset, err := staffsupport.NormalizePage(query.Page, query.PageSize)
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
	result, _, err := staffsupport.LoadCached(ctx, m.cache, &m.flights, key, staffsupport.HotReadCacheTTL, func() (Page[Room], bool, error) {
		items, total, loadErr := m.rooms.ListRooms(ctx, departmentID, offset, size)
		return Page[Room]{Items: items, Page: page, PageSize: size, Total: total}, true, loadErr
	})
	return result, err
}

// UpdateRoom 修改房间的严格地址信息，并执行版本冲突检查。
func (m *Manager) UpdateRoom(ctx context.Context, operator authn.Principal, command staffinput.UpdateRoom) (Room, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return Room{}, err
	}
	roomID, err := staffsupport.NormalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return Room{}, err
	}
	campusID, building, floorNumber, roomNumber, displayName, err := staffsupport.NormalizeRoomLocation(command.CampusID, command.Building, command.FloorNumber, command.RoomNumber)
	if err != nil {
		return Room{}, err
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return Room{}, err
	}
	if command.ExpectedVersion < 1 {
		return Room{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	command.RoomID, command.CampusID, command.Building, command.FloorNumber, command.RoomNumber, command.Operation = roomID, campusID, building, floorNumber, roomNumber, meta
	var result Room
	var departmentID string
	err = m.mutate(ctx, m.rooms, operator, meta, "room", roomID, actionRoomUpdated, command, func() string { return departmentID }, func(tx ConfigurationTxStore) (any, string, error) {
		before, lockErr := tx.GetRoomForUpdate(ctx, roomID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = before.DepartmentID
		if err := staffsupport.RequireDepartmentScope(operator, departmentID); err != nil {
			return nil, "", err
		}
		if before.Version != command.ExpectedVersion {
			return nil, "", ErrVersionConflict
		}
		if before.RetiredAt != nil {
			return nil, "", ErrInvalidState
		}
		today, weekEnd := currentBookingWeek()
		if _, err := deleteBookingsForConfiguration(ctx, tx, operator.AccountID, BookingListFilter{
			RoomID: before.RoomID, FromDate: &today, ThroughDate: &weekEnd,
		}); err != nil {
			return nil, "", err
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
	}, func(data []byte) error { return staffsupport.DecodeAfter(data, &result) })
	if departmentID == "" {
		departmentID = result.DepartmentID
	}
	if err == nil {
		m.invalidate(ctx, departmentID, "room:"+roomID)
	}
	return result, err
}

// RetireRoom 将房间标记为不再参与后续配置，同时处理受影响的预约。
func (m *Manager) RetireRoom(ctx context.Context, operator authn.Principal, command staffinput.RetireRoom) (Room, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return Room{}, err
	}
	id, err := staffsupport.NormalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return Room{}, err
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return Room{}, err
	}
	if command.ExpectedVersion < 1 {
		return Room{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	command.RoomID, command.Operation = id, meta
	var result Room
	var departmentID string
	err = m.mutate(ctx, m.rooms, operator, meta, "room", id, actionRoomRetired, command, func() string { return departmentID }, func(tx ConfigurationTxStore) (any, string, error) {
		before, lockErr := tx.GetRoomForUpdate(ctx, id)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = before.DepartmentID
		if err := staffsupport.RequireDepartmentScope(operator, departmentID); err != nil {
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
	}, func(data []byte) error { return staffsupport.DecodeAfter(data, &result) })
	if departmentID == "" {
		departmentID = result.DepartmentID
	}
	if err == nil {
		m.invalidate(ctx, departmentID, "room:"+id)
	}
	return result, err
}

// SetRoomWindow 创建或修改房间每周某天上午或下午的开放窗口和容量。
func (m *Manager) SetRoomWindow(ctx context.Context, operator authn.Principal, command staffinput.SetRoomWindow) (RoomWeeklyWindow, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return RoomWeeklyWindow{}, err
	}
	roomID, err := staffsupport.NormalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	windowID := ""
	if strings.TrimSpace(command.WindowID) != "" {
		windowID, err = staffsupport.NormalizeUUID(command.WindowID, "window_id")
		if err != nil {
			return RoomWeeklyWindow{}, err
		}
	}
	weekday, session, err := staffsupport.NormalizeSlot(command.Weekday, command.Session)
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	openTime, openMinutes, err := staffsupport.NormalizeClock(command.OpenTime, "open_time")
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	closeTime, closeMinutes, err := staffsupport.NormalizeClock(command.CloseTime, "close_time")
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	if !staffsupport.FitsSessionBoundary(session, openMinutes, closeMinutes) || command.ActiveCapacity < 1 {
		return RoomWeeklyWindow{}, fmt.Errorf("%w: room window time or capacity is invalid", ErrInvalid)
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	if windowID == "" && command.ExpectedVersion != 0 {
		return RoomWeeklyWindow{}, fmt.Errorf("%w: expected_version must be zero for a new window", ErrInvalid)
	}
	if windowID != "" && command.ExpectedVersion < 1 {
		return RoomWeeklyWindow{}, fmt.Errorf("%w: expected_version is required", ErrInvalid)
	}
	command.WindowID, command.RoomID, command.Weekday, command.Session, command.OpenTime, command.CloseTime, command.Operation = windowID, roomID, weekday, session, openTime, closeTime, meta
	var result RoomWeeklyWindow
	var departmentID string
	err = m.mutate(ctx, m.rooms, operator, meta, "room_window", windowID, actionRoomWindowSet, command, func() string { return departmentID }, func(tx ConfigurationTxStore) (any, string, error) {
		room, lockErr := tx.GetRoomForUpdate(ctx, roomID)
		if lockErr != nil {
			return nil, "", lockErr
		}
		departmentID = room.DepartmentID
		if err := staffsupport.RequireDepartmentScope(operator, departmentID); err != nil {
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
		if err := staffsupport.ValidateRoomWindowChange(ctx, tx, result); err != nil {
			return nil, "", err
		}
		if before == nil {
			if err := tx.CreateRoomWindow(ctx, result); err != nil {
				return nil, "", err
			}
		} else {
			if before.OpenTime != result.OpenTime || before.CloseTime != result.CloseTime {
				if serviceDate, relevant := currentWeekDateForWeekday(result.Weekday); relevant {
					if _, err := deleteBookingsForConfiguration(ctx, tx, operator.AccountID, BookingListFilter{
						RoomID: result.RoomID, ServiceDate: &serviceDate, Session: result.Session,
					}); err != nil {
						return nil, "", err
					}
				}
			}
			if err := reconcileRoomWindowCapacity(ctx, tx, operator.AccountID, result); err != nil {
				return nil, "", err
			}
			if err := tx.UpdateRoomWindow(ctx, result, command.ExpectedVersion); err != nil {
				return nil, "", err
			}
		}
		return struct {
			Before *RoomWeeklyWindow
			After  RoomWeeklyWindow
		}{before, result}, result.WindowID, nil
	}, func(data []byte) error { return staffsupport.DecodeAfter(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}

// ListRoomWindows 返回房间的每周开放窗口。
func (m *Manager) ListRoomWindows(ctx context.Context, operator authn.Principal, roomID string, activeOnly bool) ([]RoomWeeklyWindow, error) {
	room, err := m.GetRoom(ctx, operator, roomID)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("department:%s:g:%s:room:%s:windows:%t", room.DepartmentID, m.generation(ctx, room.DepartmentID), room.RoomID, activeOnly)
	result, _, err := staffsupport.LoadCached(ctx, m.cache, &m.flights, key, staffsupport.HotReadCacheTTL, func() ([]RoomWeeklyWindow, bool, error) {
		v, e := m.rooms.ListRoomWindows(ctx, room.RoomID, activeOnly)
		return v, true, e
	})
	return result, err
}

// DisableRoomWindow 停用房间开放窗口，并处理当前周受影响的预约。
func (m *Manager) DisableRoomWindow(ctx context.Context, operator authn.Principal, command staffinput.ChangeStatus) (RoomWeeklyWindow, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return RoomWeeklyWindow{}, err
	}
	id, err := staffsupport.NormalizeUUID(command.ResourceID, "window_id")
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	meta, err := staffinput.NormalizeOperation(command.Operation)
	if err != nil {
		return RoomWeeklyWindow{}, err
	}
	if command.ExpectedVersion < 1 {
		return RoomWeeklyWindow{}, ErrInvalid
	}
	command.ResourceID, command.Operation = id, meta
	var result RoomWeeklyWindow
	var departmentID string
	err = m.mutate(ctx, m.rooms, operator, meta, "room_window", id, actionRoomWindowDisabled, command, func() string { return departmentID }, func(tx ConfigurationTxStore) (any, string, error) {
		before, loadErr := tx.GetRoomWindowForUpdate(ctx, id)
		if loadErr != nil {
			return nil, "", loadErr
		}
		room, loadErr := tx.GetRoomForUpdate(ctx, before.RoomID)
		if loadErr != nil {
			return nil, "", loadErr
		}
		departmentID = room.DepartmentID
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
					RoomID: after.RoomID, ServiceDate: &serviceDate, Session: after.Session,
				}); deleteErr != nil {
					return nil, "", deleteErr
				}
			}
			after.Status = StatusDisabled
			after.Version++
			after.UpdatedAt = time.Now().UTC()
			if updateErr := tx.SetRoomWindowStatus(ctx, after, command.ExpectedVersion); updateErr != nil {
				return nil, "", updateErr
			}
		}
		result = after
		return struct{ Before, After RoomWeeklyWindow }{before, after}, id, nil
	}, func(data []byte) error { return staffsupport.DecodeAfter(data, &result) })
	if err == nil {
		m.invalidate(ctx, departmentID)
	}
	return result, err
}
