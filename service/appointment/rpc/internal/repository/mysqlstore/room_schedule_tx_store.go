package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager"
)

type roomScheduleTxStore struct{ *bookingTxStore }

var _ appointmentmanager.RoomScheduleTxStore = (*roomScheduleTxStore)(nil)

func (s *roomScheduleTxStore) FindOperation(ctx context.Context, operationID string) (appointmentmanager.RoomScheduleOperation, bool, error) {
	var value appointmentmanager.RoomScheduleOperation
	err := s.tx.QueryRowContext(ctx, `SELECT operator_account_id, resource_type, resource_id, action, request_fingerprint, result_data FROM appointment_resource_operations WHERE operation_id = ?`, operationID).Scan(&value.OperatorAccountID, &value.ResourceType, &value.ResourceID, &value.Action, &value.RequestFingerprint, &value.ResultData)
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.RoomScheduleOperation{}, false, nil
	}
	if err != nil {
		return appointmentmanager.RoomScheduleOperation{}, false, fmt.Errorf("find appointment resource operation: %w", err)
	}
	return value, true, nil
}

func (s *roomScheduleTxStore) RecordChange(ctx context.Context, change appointmentmanager.RoomScheduleChange) error {
	resultData, err := json.Marshal(change.After)
	if err != nil {
		return fmt.Errorf("marshal resource operation result: %w", err)
	}
	beforeData, err := marshalNullable(change.Before)
	if err != nil {
		return err
	}
	afterData, err := json.Marshal(change.After)
	if err != nil {
		return fmt.Errorf("marshal resource audit after state: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `INSERT INTO appointment_resource_operations (operation_id, operator_account_id, resource_type, resource_id, action, request_fingerprint, result_data) VALUES (?, ?, ?, ?, ?, ?, ?)`, change.OperationID, change.OperatorAccountID, change.ResourceType, change.ResourceID, change.Action, change.RequestFingerprint, resultData)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: resource operation_id already exists", appointmentmanager.ErrConflict)
		}
		return fmt.Errorf("record resource operation: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `INSERT INTO appointment_resource_audit (id, operation_id, operator_account_id, department_id, resource_type, resource_id, action, before_data, after_data, request_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))`, uuid.NewString(), change.OperationID, change.OperatorAccountID, change.DepartmentID, change.ResourceType, change.ResourceID, change.Action, beforeData, afterData, change.RequestID)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: resource audit already exists", appointmentmanager.ErrConflict)
		}
		return fmt.Errorf("record resource audit: %w", err)
	}
	return nil
}

func marshalNullable(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal resource audit before state: %w", err)
	}
	return data, nil
}

func (s *roomScheduleTxStore) GetRoomForUpdate(ctx context.Context, id string) (appointmentmanager.Room, error) {
	value, err := scanRoom(s.tx.QueryRowContext(ctx, roomSelect+" WHERE id = ? FOR UPDATE", id))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.Room{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.Room{}, fmt.Errorf("lock appointment room: %w", err)
	}
	return value, nil
}
func (s *roomScheduleTxStore) CreateRoom(ctx context.Context, v appointmentmanager.Room) error {
	_, err := s.tx.ExecContext(ctx, `INSERT INTO appointment_rooms (id, department_id, campus_id, building, floor_number, room_number, retired_at, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NULL, ?, ?, ?)`, v.RoomID, v.DepartmentID, v.CampusID, v.Building, v.FloorNumber, v.RoomNumber, v.Version, v.CreatedAt, v.UpdatedAt)
	return mapWriteError(err, "create appointment room")
}
func (s *roomScheduleTxStore) UpdateRoom(ctx context.Context, v appointmentmanager.Room, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_rooms SET campus_id = ?, building = ?, floor_number = ?, room_number = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ? AND retired_at IS NULL`, v.CampusID, v.Building, v.FloorNumber, v.RoomNumber, v.UpdatedAt, v.RoomID, expected)
	if err != nil {
		return mapWriteError(err, "update appointment room")
	}
	return s.requireMutation(ctx, result, "appointment_rooms", v.RoomID)
}
func (s *roomScheduleTxStore) RetireRoom(ctx context.Context, v appointmentmanager.Room, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_rooms SET retired_at = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ? AND retired_at IS NULL`, v.RetiredAt, v.UpdatedAt, v.RoomID, expected)
	if err != nil {
		return mapWriteError(err, "retire appointment room")
	}
	return s.requireMutation(ctx, result, "appointment_rooms", v.RoomID)
}

func (s *roomScheduleTxStore) GetItemForUpdate(ctx context.Context, id string) (appointmentmanager.ItemSummary, error) {
	var v appointmentmanager.ItemSummary
	err := s.tx.QueryRowContext(ctx, `SELECT id, owner_department_id, name, status, version FROM appointment_examination_items WHERE id = ? FOR UPDATE`, id).Scan(&v.ItemID, &v.DepartmentID, &v.Name, &v.Status, &v.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ItemSummary{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.ItemSummary{}, fmt.Errorf("lock examination item: %w", err)
	}
	return v, nil
}

func (s *roomScheduleTxStore) GetRelationForUpdate(ctx context.Context, id string) (appointmentmanager.RoomItem, error) {
	v, err := scanRelation(s.tx.QueryRowContext(ctx, relationSelect+" WHERE r.id = ? FOR UPDATE", id))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.RoomItem{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.RoomItem{}, fmt.Errorf("lock room item relation: %w", err)
	}
	return v, nil
}
func (s *roomScheduleTxStore) FindRelationForUpdate(ctx context.Context, roomID, itemID string) (appointmentmanager.RoomItem, bool, error) {
	v, err := scanRelation(s.tx.QueryRowContext(ctx, relationSelect+" WHERE r.room_id = ? AND r.item_id = ? FOR UPDATE", roomID, itemID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.RoomItem{}, false, nil
	}
	if err != nil {
		return appointmentmanager.RoomItem{}, false, fmt.Errorf("find room item relation: %w", err)
	}
	return v, true, nil
}
func (s *roomScheduleTxStore) CreateRelation(ctx context.Context, v appointmentmanager.RoomItem) error {
	_, err := s.tx.ExecContext(ctx, `INSERT INTO appointment_room_examination_items (id, room_id, item_id, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, v.RelationID, v.RoomID, v.ItemID, v.Status, v.Version, v.CreatedAt, v.UpdatedAt)
	return mapWriteError(err, "create room item relation")
}
func (s *roomScheduleTxStore) SetRelationStatus(ctx context.Context, v appointmentmanager.RoomItem, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_room_examination_items SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.Status, v.UpdatedAt, v.RelationID, expected)
	if err != nil {
		return mapWriteError(err, "set room item relation status")
	}
	return s.requireMutation(ctx, result, "appointment_room_examination_items", v.RelationID)
}
func (s *roomScheduleTxStore) ListRoomRelationsForUpdate(ctx context.Context, roomID string) ([]appointmentmanager.RoomItem, error) {
	rows, err := s.tx.QueryContext(ctx, relationSelect+" WHERE r.room_id = ? FOR UPDATE", roomID)
	if err != nil {
		return nil, fmt.Errorf("lock room relations: %w", err)
	}
	defer rows.Close()
	return scanRelations(rows)
}
func (s *roomScheduleTxStore) ListItemRelationsForUpdate(ctx context.Context, itemID string) ([]appointmentmanager.RoomItem, error) {
	rows, err := s.tx.QueryContext(ctx, relationSelect+" WHERE r.item_id = ? FOR UPDATE", itemID)
	if err != nil {
		return nil, fmt.Errorf("lock item relations: %w", err)
	}
	defer rows.Close()
	return scanRelations(rows)
}

func (s *roomScheduleTxStore) GetRoomWindowForUpdate(ctx context.Context, id string) (appointmentmanager.RoomWeeklyWindow, error) {
	v, err := scanRoomWindow(s.tx.QueryRowContext(ctx, roomWindowSelect+" WHERE id = ? FOR UPDATE", id))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.RoomWeeklyWindow{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.RoomWeeklyWindow{}, fmt.Errorf("lock room window: %w", err)
	}
	return v, nil
}
func (s *roomScheduleTxStore) FindRoomWindowForUpdate(ctx context.Context, roomID string, weekday int32, session appointmentmanager.Session) (appointmentmanager.RoomWeeklyWindow, bool, error) {
	v, err := scanRoomWindow(s.tx.QueryRowContext(ctx, roomWindowSelect+" WHERE room_id = ? AND weekday = ? AND session = ? FOR UPDATE", roomID, weekday, session))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.RoomWeeklyWindow{}, false, nil
	}
	if err != nil {
		return appointmentmanager.RoomWeeklyWindow{}, false, fmt.Errorf("find room window: %w", err)
	}
	return v, true, nil
}
func (s *roomScheduleTxStore) CreateRoomWindow(ctx context.Context, v appointmentmanager.RoomWeeklyWindow) error {
	_, err := s.tx.ExecContext(ctx, `INSERT INTO appointment_room_weekly_windows (id, room_id, weekday, session, open_time, close_time, active_capacity, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, v.WindowID, v.RoomID, v.Weekday, v.Session, v.OpenTime, v.CloseTime, v.ActiveCapacity, v.Status, v.Version, v.CreatedAt, v.UpdatedAt)
	return mapWriteError(err, "create room window")
}
func (s *roomScheduleTxStore) UpdateRoomWindow(ctx context.Context, v appointmentmanager.RoomWeeklyWindow, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_room_weekly_windows SET open_time = ?, close_time = ?, active_capacity = ?, status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.OpenTime, v.CloseTime, v.ActiveCapacity, v.Status, v.UpdatedAt, v.WindowID, expected)
	if err != nil {
		return mapWriteError(err, "update room window")
	}
	return s.requireMutation(ctx, result, "appointment_room_weekly_windows", v.WindowID)
}
func (s *roomScheduleTxStore) SetRoomWindowStatus(ctx context.Context, v appointmentmanager.RoomWeeklyWindow, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_room_weekly_windows SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.Status, v.UpdatedAt, v.WindowID, expected)
	if err != nil {
		return mapWriteError(err, "set room window status")
	}
	return s.requireMutation(ctx, result, "appointment_room_weekly_windows", v.WindowID)
}
func (s *roomScheduleTxStore) ListRoomWindowsForUpdate(ctx context.Context, roomID string) ([]appointmentmanager.RoomWeeklyWindow, error) {
	rows, err := s.tx.QueryContext(ctx, roomWindowSelect+" WHERE room_id = ? FOR UPDATE", roomID)
	if err != nil {
		return nil, fmt.Errorf("lock room windows: %w", err)
	}
	defer rows.Close()
	return scanRoomWindows(rows)
}

func (s *roomScheduleTxStore) GetItemWindowForUpdate(ctx context.Context, id string) (appointmentmanager.ItemWeeklyWindow, error) {
	v, err := scanItemWindow(s.tx.QueryRowContext(ctx, itemWindowSelect+" WHERE id = ? FOR UPDATE", id))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ItemWeeklyWindow{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.ItemWeeklyWindow{}, fmt.Errorf("lock item window: %w", err)
	}
	return v, nil
}
func (s *roomScheduleTxStore) FindItemWindowForUpdate(ctx context.Context, itemID string, weekday int32, session appointmentmanager.Session) (appointmentmanager.ItemWeeklyWindow, bool, error) {
	v, err := scanItemWindow(s.tx.QueryRowContext(ctx, itemWindowSelect+" WHERE item_id = ? AND weekday = ? AND session = ? FOR UPDATE", itemID, weekday, session))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ItemWeeklyWindow{}, false, nil
	}
	if err != nil {
		return appointmentmanager.ItemWeeklyWindow{}, false, fmt.Errorf("find item window: %w", err)
	}
	return v, true, nil
}
func (s *roomScheduleTxStore) CreateItemWindow(ctx context.Context, v appointmentmanager.ItemWeeklyWindow) error {
	_, err := s.tx.ExecContext(ctx, `INSERT INTO appointment_item_weekly_windows (id, item_id, weekday, session, start_time, booking_cutoff_time, end_time, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, v.WindowID, v.ItemID, v.Weekday, v.Session, v.StartTime, v.BookingCutoffTime, v.EndTime, v.Status, v.Version, v.CreatedAt, v.UpdatedAt)
	return mapWriteError(err, "create item window")
}
func (s *roomScheduleTxStore) UpdateItemWindow(ctx context.Context, v appointmentmanager.ItemWeeklyWindow, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_item_weekly_windows SET start_time = ?, booking_cutoff_time = ?, end_time = ?, status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.StartTime, v.BookingCutoffTime, v.EndTime, v.Status, v.UpdatedAt, v.WindowID, expected)
	if err != nil {
		return mapWriteError(err, "update item window")
	}
	return s.requireMutation(ctx, result, "appointment_item_weekly_windows", v.WindowID)
}
func (s *roomScheduleTxStore) SetItemWindowStatus(ctx context.Context, v appointmentmanager.ItemWeeklyWindow, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_item_weekly_windows SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.Status, v.UpdatedAt, v.WindowID, expected)
	if err != nil {
		return mapWriteError(err, "set item window status")
	}
	return s.requireMutation(ctx, result, "appointment_item_weekly_windows", v.WindowID)
}
func (s *roomScheduleTxStore) ListItemWindowsForUpdate(ctx context.Context, itemID string) ([]appointmentmanager.ItemWeeklyWindow, error) {
	rows, err := s.tx.QueryContext(ctx, itemWindowSelect+" WHERE item_id = ? FOR UPDATE", itemID)
	if err != nil {
		return nil, fmt.Errorf("lock item windows: %w", err)
	}
	defer rows.Close()
	return scanItemWindows(rows)
}

func mapWriteError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if isDuplicateEntry(err) {
		return fmt.Errorf("%w: duplicate appointment resource", appointmentmanager.ErrConflict)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
func (s *roomScheduleTxStore) requireMutation(ctx context.Context, result sql.Result, table, id string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected > 0 {
		return nil
	}
	allowed := map[string]bool{"appointment_rooms": true, "appointment_room_examination_items": true, "appointment_room_weekly_windows": true, "appointment_item_weekly_windows": true}
	if !allowed[table] {
		return fmt.Errorf("unsupported resource table")
	}
	var exists bool
	if err = s.tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM "+table+" WHERE id = ?)", id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return appointmentmanager.ErrNotFound
	}
	return appointmentmanager.ErrVersionConflict
}
