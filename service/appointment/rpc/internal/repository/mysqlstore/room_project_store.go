package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
)

const roomSelect = `SELECT id, department_id, campus_id, building, floor_number, room_number, retired_at, version, created_at, updated_at FROM appointment_rooms`
const relationSelect = `
SELECT r.id, r.room_id, r.item_id, room.campus_id, room.building, room.floor_number, room.room_number,
       item.name, r.status, r.version, r.created_at, r.updated_at
FROM appointment_room_examination_items r
JOIN appointment_rooms room ON room.id = r.room_id
JOIN appointment_examination_items item ON item.id = r.item_id`
const roomWindowSelect = `
SELECT id, room_id, weekday, session, TIME_FORMAT(open_time, '%H:%i'), TIME_FORMAT(close_time, '%H:%i'),
       active_capacity, status, version, created_at, updated_at
FROM appointment_room_weekly_windows`
const itemWindowSelect = `
SELECT id, item_id, weekday, session, TIME_FORMAT(start_time, '%H:%i'),
       TIME_FORMAT(booking_cutoff_time, '%H:%i'), TIME_FORMAT(end_time, '%H:%i'),
       status, version, created_at, updated_at
FROM appointment_item_weekly_windows`

func (s *Store) GetRoom(ctx context.Context, roomID string) (appointmentmanager.Room, error) {
	value, err := scanRoom(s.db.QueryRowContext(ctx, roomSelect+" WHERE id = ?", roomID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.Room{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.Room{}, fmt.Errorf("get appointment room: %w", err)
	}
	return value, nil
}

func (s *Store) ListRooms(ctx context.Context, departmentID string, offset, limit int64) ([]appointmentmanager.Room, int64, error) {
	where, args := " WHERE department_id = ? AND retired_at IS NULL", []any{departmentID}
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM appointment_rooms"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count appointment rooms: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, roomSelect+where+" ORDER BY building, floor_number, room_number, id LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list appointment rooms: %w", err)
	}
	defer rows.Close()
	values := make([]appointmentmanager.Room, 0)
	for rows.Next() {
		value, e := scanRoom(rows)
		if e != nil {
			return nil, 0, fmt.Errorf("scan appointment room: %w", e)
		}
		values = append(values, value)
	}
	return values, total, rows.Err()
}

func (s *Store) GetItemSummary(ctx context.Context, itemID string) (appointmentmanager.ItemSummary, error) {
	var value appointmentmanager.ItemSummary
	err := s.db.QueryRowContext(ctx, `SELECT id, owner_department_id, name, status, version FROM appointment_examination_items WHERE id = ?`, itemID).Scan(&value.ItemID, &value.DepartmentID, &value.Name, &value.Status, &value.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ItemSummary{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.ItemSummary{}, fmt.Errorf("get examination item summary: %w", err)
	}
	return value, nil
}

func (s *Store) ListRoomItems(ctx context.Context, roomID string, status appointmentmanager.Status, offset, limit int64) ([]appointmentmanager.RoomItem, int64, error) {
	where, args := " WHERE r.room_id = ?", []any{roomID}
	if status != "" {
		where += " AND r.status = ?"
		args = append(args, status)
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM appointment_room_examination_items r"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count room examination items: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, relationSelect+where+" ORDER BY item.name, r.id LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list room examination items: %w", err)
	}
	defer rows.Close()
	values, err := scanRelations(rows)
	return values, total, err
}

func (s *Store) ListItemRooms(ctx context.Context, itemID string, activeOnly bool) ([]appointmentmanager.RoomItem, error) {
	where, args := " WHERE r.item_id = ?", []any{itemID}
	if activeOnly {
		where += " AND r.status = 'active' AND room.retired_at IS NULL"
	}
	rows, err := s.db.QueryContext(ctx, relationSelect+where+" ORDER BY room.building, room.floor_number, room.room_number, r.id", args...)
	if err != nil {
		return nil, fmt.Errorf("list item rooms: %w", err)
	}
	defer rows.Close()
	return scanRelations(rows)
}

func (s *Store) ListRoomWindows(ctx context.Context, roomID string, activeOnly bool) ([]appointmentmanager.RoomWeeklyWindow, error) {
	where := " WHERE room_id = ?"
	if activeOnly {
		where += " AND status = 'active'"
	}
	rows, err := s.db.QueryContext(ctx, roomWindowSelect+where+" ORDER BY weekday, FIELD(session, 'morning', 'afternoon')", roomID)
	if err != nil {
		return nil, fmt.Errorf("list room windows: %w", err)
	}
	defer rows.Close()
	return scanRoomWindows(rows)
}
func (s *Store) ListItemWindows(ctx context.Context, itemID string, activeOnly bool) ([]appointmentmanager.ItemWeeklyWindow, error) {
	where := " WHERE item_id = ?"
	if activeOnly {
		where += " AND status = 'active'"
	}
	rows, err := s.db.QueryContext(ctx, itemWindowSelect+where+" ORDER BY weekday, FIELD(session, 'morning', 'afternoon')", itemID)
	if err != nil {
		return nil, fmt.Errorf("list item windows: %w", err)
	}
	defer rows.Close()
	return scanItemWindows(rows)
}

func (s *Store) WithinConfigurationTransaction(ctx context.Context, fn func(appointmentmanager.ConfigurationTxStore) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin room and project configuration transaction: %w", err)
	}
	if err = fn(&configurationTxStore{bookingTxStore: &bookingTxStore{tx: tx}}); err != nil {
		if rb := tx.Rollback(); rb != nil && !errors.Is(rb, sql.ErrTxDone) {
			return errors.Join(err, rb)
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit room and project configuration transaction: %w", err)
	}
	return nil
}

var (
	_ staffmanager.ProjectStore = (*Store)(nil)
	_ staffmanager.RoomStore    = (*Store)(nil)
)

type rowScanner interface{ Scan(...any) error }

func scanRoom(s rowScanner) (appointmentmanager.Room, error) {
	var v appointmentmanager.Room
	err := s.Scan(&v.RoomID, &v.DepartmentID, &v.CampusID, &v.Building, &v.FloorNumber, &v.RoomNumber, &v.RetiredAt, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	v.DisplayName = appointmentmanager.FormatRoomDisplayName(v.Building, v.FloorNumber, v.RoomNumber)
	return v, err
}
func scanRelation(s rowScanner) (appointmentmanager.RoomItem, error) {
	var v appointmentmanager.RoomItem
	err := s.Scan(&v.RelationID, &v.RoomID, &v.ItemID, &v.CampusID, &v.Building, &v.FloorNumber, &v.RoomNumber, &v.ItemName, &v.Status, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	v.RoomDisplayName = appointmentmanager.FormatRoomDisplayName(v.Building, v.FloorNumber, v.RoomNumber)
	return v, err
}
func scanRoomWindow(s rowScanner) (appointmentmanager.RoomWeeklyWindow, error) {
	var v appointmentmanager.RoomWeeklyWindow
	err := s.Scan(&v.WindowID, &v.RoomID, &v.Weekday, &v.Session, &v.OpenTime, &v.CloseTime, &v.ActiveCapacity, &v.Status, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func scanItemWindow(s rowScanner) (appointmentmanager.ItemWeeklyWindow, error) {
	var v appointmentmanager.ItemWeeklyWindow
	err := s.Scan(&v.WindowID, &v.ItemID, &v.Weekday, &v.Session, &v.StartTime, &v.BookingCutoffTime, &v.EndTime, &v.Status, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func scanRelations(rows *sql.Rows) ([]appointmentmanager.RoomItem, error) {
	values := make([]appointmentmanager.RoomItem, 0)
	for rows.Next() {
		v, err := scanRelation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan room item relation: %w", err)
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
func scanRoomWindows(rows *sql.Rows) ([]appointmentmanager.RoomWeeklyWindow, error) {
	values := make([]appointmentmanager.RoomWeeklyWindow, 0)
	for rows.Next() {
		v, err := scanRoomWindow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan room window: %w", err)
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
func scanItemWindows(rows *sql.Rows) ([]appointmentmanager.ItemWeeklyWindow, error) {
	values := make([]appointmentmanager.ItemWeeklyWindow, 0)
	for rows.Next() {
		v, err := scanItemWindow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan item window: %w", err)
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func statusWhere(alias string, status appointmentmanager.Status) (string, []any) {
	if status == "" {
		return "", nil
	}
	return " AND " + strings.TrimSpace(alias) + "status = ?", []any{status}
}
