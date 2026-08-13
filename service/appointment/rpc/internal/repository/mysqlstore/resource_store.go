package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"hospital/service/appointment/rpc/internal/resource"
)

const roomSelect = `SELECT id, department_id, name, status, version, created_at, updated_at FROM appointment_rooms`
const relationSelect = `
SELECT r.id, r.room_id, r.item_id, room.name, item.name, r.status, r.version, r.created_at, r.updated_at
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

func (s *Store) GetRoom(ctx context.Context, roomID string) (resource.Room, error) {
	value, err := scanRoom(s.db.QueryRowContext(ctx, roomSelect+" WHERE id = ?", roomID))
	if errors.Is(err, sql.ErrNoRows) {
		return resource.Room{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.Room{}, fmt.Errorf("get appointment room: %w", err)
	}
	return value, nil
}

func (s *Store) ListRooms(ctx context.Context, departmentID string, status resource.Status, offset, limit int64) ([]resource.Room, int64, error) {
	where, args := " WHERE department_id = ?", []any{departmentID}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM appointment_rooms"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count appointment rooms: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, roomSelect+where+" ORDER BY name, id LIMIT ? OFFSET ?", append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list appointment rooms: %w", err)
	}
	defer rows.Close()
	values := make([]resource.Room, 0)
	for rows.Next() {
		value, e := scanRoom(rows)
		if e != nil {
			return nil, 0, fmt.Errorf("scan appointment room: %w", e)
		}
		values = append(values, value)
	}
	return values, total, rows.Err()
}

func (s *Store) GetItemSummary(ctx context.Context, itemID string) (resource.ItemSummary, error) {
	var value resource.ItemSummary
	err := s.db.QueryRowContext(ctx, `SELECT id, owner_department_id, name, status, version FROM appointment_examination_items WHERE id = ?`, itemID).Scan(&value.ItemID, &value.DepartmentID, &value.Name, &value.Status, &value.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return resource.ItemSummary{}, resource.ErrNotFound
	}
	if err != nil {
		return resource.ItemSummary{}, fmt.Errorf("get examination item summary: %w", err)
	}
	return value, nil
}

func (s *Store) ListRoomItems(ctx context.Context, roomID string, status resource.Status, offset, limit int64) ([]resource.RoomItem, int64, error) {
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

func (s *Store) ListItemRooms(ctx context.Context, itemID string, activeOnly bool) ([]resource.RoomItem, error) {
	where, args := " WHERE r.item_id = ?", []any{itemID}
	if activeOnly {
		where += " AND r.status = 'active' AND room.status = 'active'"
	}
	rows, err := s.db.QueryContext(ctx, relationSelect+where+" ORDER BY room.name, r.id", args...)
	if err != nil {
		return nil, fmt.Errorf("list item rooms: %w", err)
	}
	defer rows.Close()
	return scanRelations(rows)
}

func (s *Store) ListRoomWindows(ctx context.Context, roomID string, activeOnly bool) ([]resource.RoomWeeklyWindow, error) {
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
func (s *Store) ListItemWindows(ctx context.Context, itemID string, activeOnly bool) ([]resource.ItemWeeklyWindow, error) {
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

func (s *Store) WithinResourceTransaction(ctx context.Context, fn func(resource.TxStore) error) error {
	if fn == nil {
		return errors.New("resource transaction callback is required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin resource transaction: %w", err)
	}
	if err = fn(&resourceTxStore{tx: tx}); err != nil {
		if rb := tx.Rollback(); rb != nil && !errors.Is(rb, sql.ErrTxDone) {
			return errors.Join(err, rb)
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit resource transaction: %w", err)
	}
	return nil
}

var _ resource.Store = (*Store)(nil)

type rowScanner interface{ Scan(...any) error }

func scanRoom(s rowScanner) (resource.Room, error) {
	var v resource.Room
	err := s.Scan(&v.RoomID, &v.DepartmentID, &v.Name, &v.Status, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func scanRelation(s rowScanner) (resource.RoomItem, error) {
	var v resource.RoomItem
	err := s.Scan(&v.RelationID, &v.RoomID, &v.ItemID, &v.RoomName, &v.ItemName, &v.Status, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func scanRoomWindow(s rowScanner) (resource.RoomWeeklyWindow, error) {
	var v resource.RoomWeeklyWindow
	err := s.Scan(&v.WindowID, &v.RoomID, &v.Weekday, &v.Session, &v.OpenTime, &v.CloseTime, &v.ActiveCapacity, &v.Status, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func scanItemWindow(s rowScanner) (resource.ItemWeeklyWindow, error) {
	var v resource.ItemWeeklyWindow
	err := s.Scan(&v.WindowID, &v.ItemID, &v.Weekday, &v.Session, &v.StartTime, &v.BookingCutoffTime, &v.EndTime, &v.Status, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}
func scanRelations(rows *sql.Rows) ([]resource.RoomItem, error) {
	values := make([]resource.RoomItem, 0)
	for rows.Next() {
		v, err := scanRelation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan room item relation: %w", err)
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
func scanRoomWindows(rows *sql.Rows) ([]resource.RoomWeeklyWindow, error) {
	values := make([]resource.RoomWeeklyWindow, 0)
	for rows.Next() {
		v, err := scanRoomWindow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan room window: %w", err)
		}
		values = append(values, v)
	}
	return values, rows.Err()
}
func scanItemWindows(rows *sql.Rows) ([]resource.ItemWeeklyWindow, error) {
	values := make([]resource.ItemWeeklyWindow, 0)
	for rows.Next() {
		v, err := scanItemWindow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan item window: %w", err)
		}
		values = append(values, v)
	}
	return values, rows.Err()
}

func statusWhere(alias string, status resource.Status) (string, []any) {
	if status == "" {
		return "", nil
	}
	return " AND " + strings.TrimSpace(alias) + "status = ?", []any{status}
}
