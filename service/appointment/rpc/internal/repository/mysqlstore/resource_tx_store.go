package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"hospital/service/appointment/rpc/internal/resource"
)

type resourceTxStore struct{ tx *sql.Tx }

var _ resource.TxStore = (*resourceTxStore)(nil)

func (s *resourceTxStore) FindOperation(ctx context.Context, operationID string) (resource.Operation, bool, error) {
	var value resource.Operation
	err := s.tx.QueryRowContext(ctx, `SELECT operator_account_id, resource_type, resource_id, action, request_fingerprint, result_data FROM appointment_resource_operations WHERE operation_id = ?`, operationID).Scan(&value.OperatorAccountID, &value.ResourceType, &value.ResourceID, &value.Action, &value.RequestFingerprint, &value.ResultData)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.Operation{}, false, nil
	}
	if err != nil {
		return resource.Operation{}, false, fmt.Errorf("find appointment resource operation: %w", err)
	}
	return value, true, nil
}

func (s *resourceTxStore) RecordChange(ctx context.Context, change resource.Change) error {
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
			return fmt.Errorf("%w: resource operation_id already exists", resource.ErrConflict)
		}
		return fmt.Errorf("record resource operation: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `INSERT INTO appointment_resource_audit (id, operation_id, operator_account_id, department_id, resource_type, resource_id, action, before_data, after_data, request_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))`, uuid.NewString(), change.OperationID, change.OperatorAccountID, change.DepartmentID, change.ResourceType, change.ResourceID, change.Action, beforeData, afterData, change.RequestID)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: resource audit already exists", resource.ErrConflict)
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

func (s *resourceTxStore) GetRoomForUpdate(ctx context.Context, id string) (resource.Room, error) {
	value, err := scanRoom(s.tx.QueryRowContext(ctx, roomSelect+" WHERE id = ? FOR UPDATE", id))
	if errors.Is(err, sql.ErrNoRows) {
		return resource.Room{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.Room{}, fmt.Errorf("lock appointment room: %w", err)
	}
	return value, nil
}
func (s *resourceTxStore) CreateRoom(ctx context.Context, v resource.Room) error {
	_, err := s.tx.ExecContext(ctx, `INSERT INTO appointment_rooms (id, department_id, name, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, v.RoomID, v.DepartmentID, v.Name, v.Status, v.Version, v.CreatedAt, v.UpdatedAt)
	return mapWriteError(err, "create appointment room")
}
func (s *resourceTxStore) UpdateRoom(ctx context.Context, v resource.Room, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_rooms SET name = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.Name, v.UpdatedAt, v.RoomID, expected)
	if err != nil {
		return mapWriteError(err, "update appointment room")
	}
	return s.requireMutation(ctx, result, "appointment_rooms", v.RoomID)
}
func (s *resourceTxStore) SetRoomStatus(ctx context.Context, v resource.Room, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_rooms SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.Status, v.UpdatedAt, v.RoomID, expected)
	if err != nil {
		return mapWriteError(err, "set appointment room status")
	}
	return s.requireMutation(ctx, result, "appointment_rooms", v.RoomID)
}

func (s *resourceTxStore) GetItemForUpdate(ctx context.Context, id string) (resource.ItemSummary, error) {
	var v resource.ItemSummary
	err := s.tx.QueryRowContext(ctx, `SELECT id, owner_department_id, name, status, version FROM appointment_examination_items WHERE id = ? FOR UPDATE`, id).Scan(&v.ItemID, &v.DepartmentID, &v.Name, &v.Status, &v.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.ItemSummary{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.ItemSummary{}, fmt.Errorf("lock examination item: %w", err)
	}
	return v, nil
}

func (s *resourceTxStore) GetRelationForUpdate(ctx context.Context, id string) (resource.RoomItem, error) {
	v, err := scanRelation(s.tx.QueryRowContext(ctx, relationSelect+" WHERE r.id = ? FOR UPDATE", id))
	if errors.Is(err, sql.ErrNoRows) {
		return resource.RoomItem{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.RoomItem{}, fmt.Errorf("lock room item relation: %w", err)
	}
	return v, nil
}
func (s *resourceTxStore) FindRelationForUpdate(ctx context.Context, roomID, itemID string) (resource.RoomItem, bool, error) {
	v, err := scanRelation(s.tx.QueryRowContext(ctx, relationSelect+" WHERE r.room_id = ? AND r.item_id = ? FOR UPDATE", roomID, itemID))
	if errors.Is(err, sql.ErrNoRows) {
		return resource.RoomItem{}, false, nil
	}
	if err != nil {
		return resource.RoomItem{}, false, fmt.Errorf("find room item relation: %w", err)
	}
	return v, true, nil
}
func (s *resourceTxStore) CreateRelation(ctx context.Context, v resource.RoomItem) error {
	_, err := s.tx.ExecContext(ctx, `INSERT INTO appointment_room_examination_items (id, room_id, item_id, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, v.RelationID, v.RoomID, v.ItemID, v.Status, v.Version, v.CreatedAt, v.UpdatedAt)
	return mapWriteError(err, "create room item relation")
}
func (s *resourceTxStore) SetRelationStatus(ctx context.Context, v resource.RoomItem, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_room_examination_items SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.Status, v.UpdatedAt, v.RelationID, expected)
	if err != nil {
		return mapWriteError(err, "set room item relation status")
	}
	return s.requireMutation(ctx, result, "appointment_room_examination_items", v.RelationID)
}
func (s *resourceTxStore) ListRoomRelationsForUpdate(ctx context.Context, roomID string) ([]resource.RoomItem, error) {
	rows, err := s.tx.QueryContext(ctx, relationSelect+" WHERE r.room_id = ? FOR UPDATE", roomID)
	if err != nil {
		return nil, fmt.Errorf("lock room relations: %w", err)
	}
	defer rows.Close()
	return scanRelations(rows)
}
func (s *resourceTxStore) ListItemRelationsForUpdate(ctx context.Context, itemID string) ([]resource.RoomItem, error) {
	rows, err := s.tx.QueryContext(ctx, relationSelect+" WHERE r.item_id = ? FOR UPDATE", itemID)
	if err != nil {
		return nil, fmt.Errorf("lock item relations: %w", err)
	}
	defer rows.Close()
	return scanRelations(rows)
}

func (s *resourceTxStore) GetRoomWindowForUpdate(ctx context.Context, id string) (resource.RoomWeeklyWindow, error) {
	v, err := scanRoomWindow(s.tx.QueryRowContext(ctx, roomWindowSelect+" WHERE id = ? FOR UPDATE", id))
	if errors.Is(err, sql.ErrNoRows) {
		return resource.RoomWeeklyWindow{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.RoomWeeklyWindow{}, fmt.Errorf("lock room window: %w", err)
	}
	return v, nil
}
func (s *resourceTxStore) FindRoomWindowForUpdate(ctx context.Context, roomID string, weekday int32, session resource.Session) (resource.RoomWeeklyWindow, bool, error) {
	v, err := scanRoomWindow(s.tx.QueryRowContext(ctx, roomWindowSelect+" WHERE room_id = ? AND weekday = ? AND session = ? FOR UPDATE", roomID, weekday, session))
	if errors.Is(err, sql.ErrNoRows) {
		return resource.RoomWeeklyWindow{}, false, nil
	}
	if err != nil {
		return resource.RoomWeeklyWindow{}, false, fmt.Errorf("find room window: %w", err)
	}
	return v, true, nil
}
func (s *resourceTxStore) CreateRoomWindow(ctx context.Context, v resource.RoomWeeklyWindow) error {
	_, err := s.tx.ExecContext(ctx, `INSERT INTO appointment_room_weekly_windows (id, room_id, weekday, session, open_time, close_time, active_capacity, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, v.WindowID, v.RoomID, v.Weekday, v.Session, v.OpenTime, v.CloseTime, v.ActiveCapacity, v.Status, v.Version, v.CreatedAt, v.UpdatedAt)
	return mapWriteError(err, "create room window")
}
func (s *resourceTxStore) UpdateRoomWindow(ctx context.Context, v resource.RoomWeeklyWindow, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_room_weekly_windows SET open_time = ?, close_time = ?, active_capacity = ?, status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.OpenTime, v.CloseTime, v.ActiveCapacity, v.Status, v.UpdatedAt, v.WindowID, expected)
	if err != nil {
		return mapWriteError(err, "update room window")
	}
	return s.requireMutation(ctx, result, "appointment_room_weekly_windows", v.WindowID)
}
func (s *resourceTxStore) SetRoomWindowStatus(ctx context.Context, v resource.RoomWeeklyWindow, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_room_weekly_windows SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.Status, v.UpdatedAt, v.WindowID, expected)
	if err != nil {
		return mapWriteError(err, "set room window status")
	}
	return s.requireMutation(ctx, result, "appointment_room_weekly_windows", v.WindowID)
}
func (s *resourceTxStore) ListRoomWindowsForUpdate(ctx context.Context, roomID string) ([]resource.RoomWeeklyWindow, error) {
	rows, err := s.tx.QueryContext(ctx, roomWindowSelect+" WHERE room_id = ? FOR UPDATE", roomID)
	if err != nil {
		return nil, fmt.Errorf("lock room windows: %w", err)
	}
	defer rows.Close()
	return scanRoomWindows(rows)
}

func (s *resourceTxStore) GetItemWindowForUpdate(ctx context.Context, id string) (resource.ItemWeeklyWindow, error) {
	v, err := scanItemWindow(s.tx.QueryRowContext(ctx, itemWindowSelect+" WHERE id = ? FOR UPDATE", id))
	if errors.Is(err, sql.ErrNoRows) {
		return resource.ItemWeeklyWindow{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.ItemWeeklyWindow{}, fmt.Errorf("lock item window: %w", err)
	}
	return v, nil
}
func (s *resourceTxStore) FindItemWindowForUpdate(ctx context.Context, itemID string, weekday int32, session resource.Session) (resource.ItemWeeklyWindow, bool, error) {
	v, err := scanItemWindow(s.tx.QueryRowContext(ctx, itemWindowSelect+" WHERE item_id = ? AND weekday = ? AND session = ? FOR UPDATE", itemID, weekday, session))
	if errors.Is(err, sql.ErrNoRows) {
		return resource.ItemWeeklyWindow{}, false, nil
	}
	if err != nil {
		return resource.ItemWeeklyWindow{}, false, fmt.Errorf("find item window: %w", err)
	}
	return v, true, nil
}
func (s *resourceTxStore) CreateItemWindow(ctx context.Context, v resource.ItemWeeklyWindow) error {
	_, err := s.tx.ExecContext(ctx, `INSERT INTO appointment_item_weekly_windows (id, item_id, weekday, session, start_time, booking_cutoff_time, end_time, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, v.WindowID, v.ItemID, v.Weekday, v.Session, v.StartTime, v.BookingCutoffTime, v.EndTime, v.Status, v.Version, v.CreatedAt, v.UpdatedAt)
	return mapWriteError(err, "create item window")
}
func (s *resourceTxStore) UpdateItemWindow(ctx context.Context, v resource.ItemWeeklyWindow, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_item_weekly_windows SET start_time = ?, booking_cutoff_time = ?, end_time = ?, status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.StartTime, v.BookingCutoffTime, v.EndTime, v.Status, v.UpdatedAt, v.WindowID, expected)
	if err != nil {
		return mapWriteError(err, "update item window")
	}
	return s.requireMutation(ctx, result, "appointment_item_weekly_windows", v.WindowID)
}
func (s *resourceTxStore) SetItemWindowStatus(ctx context.Context, v resource.ItemWeeklyWindow, expected int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_item_weekly_windows SET status = ?, version = version + 1, updated_at = ? WHERE id = ? AND version = ?`, v.Status, v.UpdatedAt, v.WindowID, expected)
	if err != nil {
		return mapWriteError(err, "set item window status")
	}
	return s.requireMutation(ctx, result, "appointment_item_weekly_windows", v.WindowID)
}
func (s *resourceTxStore) ListItemWindowsForUpdate(ctx context.Context, itemID string) ([]resource.ItemWeeklyWindow, error) {
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
		return fmt.Errorf("%w: duplicate appointment resource", resource.ErrConflict)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
func (s *resourceTxStore) requireMutation(ctx context.Context, result sql.Result, table, id string) error {
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
		return resource.ErrNotFound
	}
	return resource.ErrVersionConflict
}
