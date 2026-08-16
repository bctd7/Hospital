package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
	patientmanager "hospital/service/appointment/rpc/internal/manager/patient"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
)

const bookingSelect = `
SELECT b.id, b.patient_account_id, b.patient_display_name_snapshot,
       b.patient_phone_masked_snapshot, b.patient_phone_last4_snapshot,
       b.department_id, b.item_id, i.name,
       b.room_id, r.campus_id, r.building, r.floor_number, r.room_number,
       b.service_date, b.session, b.status,
       TIME_FORMAT(b.room_open_time_snapshot, '%H:%i:%s'),
       TIME_FORMAT(b.room_close_time_snapshot, '%H:%i:%s'),
       TIME_FORMAT(b.item_start_time_snapshot, '%H:%i:%s'),
       TIME_FORMAT(b.item_end_time_snapshot, '%H:%i:%s'),
       TIME_FORMAT(b.item_cutoff_time_snapshot, '%H:%i:%s'),
       b.estimated_duration_minutes_snapshot,
       b.started_at, COALESCE(b.started_by, ''), COALESCE(b.started_by_display_name_snapshot, ''),
       b.examination_ended_at, COALESCE(b.examination_ended_by, ''), COALESCE(b.examination_ended_by_display_name_snapshot, ''),
       b.completed_at, COALESCE(b.completed_by, ''), COALESCE(b.completed_by_display_name_snapshot, ''),
       COALESCE(qe.ticket_number, 0), COALESCE(current_qe.ticket_number, 0),
       qe.checked_in_at, qe.called_at, qe.call_deadline, COALESCE(qe.call_attempts, 0),
       CASE WHEN qe.booking_id IS NULL THEN 0 ELSE (
         SELECT COUNT(*)
         FROM appointment_check_queue_entries ahead
         JOIN appointment_bookings ahead_booking ON ahead_booking.id = ahead.booking_id AND ahead_booking.status = 'queued'
         WHERE ahead.queue_id = qe.queue_id AND ahead.booking_id <> b.id
           AND (
             (ahead.eligible_after_call_sequence <= q.call_sequence AND qe.eligible_after_call_sequence > q.call_sequence)
             OR (ahead.eligible_after_call_sequence <= q.call_sequence AND qe.eligible_after_call_sequence <= q.call_sequence AND ahead.ticket_number < qe.ticket_number)
             OR (ahead.eligible_after_call_sequence > q.call_sequence AND qe.eligible_after_call_sequence > q.call_sequence
                 AND (ahead.eligible_after_call_sequence < qe.eligible_after_call_sequence
                      OR (ahead.eligible_after_call_sequence = qe.eligible_after_call_sequence AND ahead.ticket_number < qe.ticket_number)))
           )
       ) END,
       b.version, b.created_at, b.updated_at
FROM appointment_bookings b
JOIN appointment_examination_items i ON i.id = b.item_id
JOIN appointment_rooms r ON r.id = b.room_id
LEFT JOIN appointment_check_queue_entries qe ON qe.booking_id = b.id
LEFT JOIN appointment_check_queues q ON q.id = qe.queue_id
LEFT JOIN appointment_check_queue_entries current_qe ON current_qe.booking_id = q.current_called_booking_id`

func (s *Store) ListBookingOptions(ctx context.Context, itemID string, fromDate, throughDate time.Time) ([]appointmentmanager.BookingOption, error) {
	rows, err := s.db.QueryContext(ctx, `
WITH RECURSIVE booking_dates(service_date) AS (
    SELECT DATE(?)
    UNION ALL
    SELECT DATE_ADD(service_date, INTERVAL 1 DAY)
    FROM booking_dates
    WHERE service_date < DATE(?)
)
SELECT i.id, i.owner_department_id, i.estimated_duration_minutes,
       r.id, r.campus_id, r.building, r.floor_number, r.room_number,
       d.service_date, iw.session,
       TIME_FORMAT(rw.open_time, '%H:%i:%s'), TIME_FORMAT(rw.close_time, '%H:%i:%s'),
       TIME_FORMAT(iw.start_time, '%H:%i:%s'), TIME_FORMAT(iw.end_time, '%H:%i:%s'),
       TIME_FORMAT(iw.booking_cutoff_time, '%H:%i:%s'),
       rw.active_capacity, COALESCE(c.occupied_capacity, 0)
FROM appointment_examination_items i
JOIN appointment_room_examination_items ri
  ON ri.item_id = i.id AND ri.status = 'active'
JOIN appointment_rooms r
  ON r.id = ri.room_id AND r.retired_at IS NULL
JOIN booking_dates d
JOIN appointment_item_weekly_windows iw
  ON iw.item_id = i.id
 AND iw.weekday = WEEKDAY(d.service_date) + 1
 AND iw.status = 'active'
JOIN appointment_room_weekly_windows rw
  ON rw.room_id = r.id
 AND rw.weekday = iw.weekday
 AND rw.session = iw.session
 AND rw.status = 'active'
LEFT JOIN appointment_room_date_capacity c
  ON c.room_id = r.id
 AND c.service_date = d.service_date
 AND c.session = iw.session
WHERE i.id = ?
  AND i.status = 'active'
  AND rw.open_time <= iw.start_time
  AND iw.end_time <= rw.close_time
ORDER BY d.service_date, iw.session, r.building, r.floor_number, r.room_number, r.id`,
		fromDate.Format("2006-01-02"), throughDate.Format("2006-01-02"), itemID,
	)
	if err != nil {
		return nil, fmt.Errorf("list booking options: %w", err)
	}
	defer rows.Close()

	options := make([]appointmentmanager.BookingOption, 0)
	for rows.Next() {
		var value appointmentmanager.BookingOption
		if err := rows.Scan(
			&value.ItemID, &value.DepartmentID, &value.EstimatedDurationMinutes,
			&value.RoomID, &value.CampusID, &value.Building, &value.FloorNumber, &value.RoomNumber,
			&value.ServiceDate, &value.Session,
			&value.RoomOpenTime, &value.RoomCloseTime,
			&value.ItemStartTime, &value.ItemEndTime, &value.BookingCutoffTime,
			&value.TotalCapacity, &value.OccupiedCapacity,
		); err != nil {
			return nil, fmt.Errorf("scan booking option: %w", err)
		}
		value.RoomDisplayName = appointmentmanager.FormatRoomDisplayName(value.Building, value.FloorNumber, value.RoomNumber)
		value.RemainingCapacity = value.TotalCapacity - value.OccupiedCapacity
		if value.RemainingCapacity < 0 {
			value.RemainingCapacity = 0
		}
		options = append(options, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate booking options: %w", err)
	}
	return options, nil
}

func (s *Store) GetBooking(ctx context.Context, bookingID string) (appointmentmanager.Booking, error) {
	value, err := scanBooking(s.db.QueryRowContext(ctx, bookingSelect+" WHERE b.id = ?", bookingID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.Booking{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.Booking{}, fmt.Errorf("get booking: %w", err)
	}
	return value, nil
}

func (s *Store) ListBookings(ctx context.Context, filter appointmentmanager.BookingListFilter) ([]appointmentmanager.Booking, int64, error) {
	conditions := make([]string, 0, 7)
	args := make([]any, 0, 9)
	appendCondition := func(condition string, value any) {
		conditions = append(conditions, condition)
		args = append(args, value)
	}
	if filter.PatientAccountID != "" {
		appendCondition("b.patient_account_id = ?", filter.PatientAccountID)
	}
	if filter.PatientKeyword != "" {
		keyword := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filter.PatientKeyword, `\`, `\\`), `%`, `\%`), `_`, `\_`)
		conditions = append(conditions, "(b.patient_display_name_snapshot LIKE ? ESCAPE '\\\\' OR b.patient_phone_last4_snapshot = ?)")
		args = append(args, "%"+keyword+"%", filter.PatientKeyword)
	}
	if filter.DepartmentID != "" {
		appendCondition("b.department_id = ?", filter.DepartmentID)
	}
	if filter.ServiceDate != nil {
		appendCondition("b.service_date = ?", filter.ServiceDate.Format("2006-01-02"))
	}
	if filter.FromDate != nil {
		appendCondition("b.service_date >= ?", filter.FromDate.Format("2006-01-02"))
	}
	if filter.ThroughDate != nil {
		appendCondition("b.service_date <= ?", filter.ThroughDate.Format("2006-01-02"))
	}
	if filter.Session != "" {
		appendCondition("b.session = ?", filter.Session)
	}
	if filter.ItemID != "" {
		appendCondition("b.item_id = ?", filter.ItemID)
	}
	if filter.RoomID != "" {
		appendCondition("b.room_id = ?", filter.RoomID)
	}
	if filter.Status != "" {
		appendCondition("b.status = ?", filter.Status)
	}
	switch filter.View {
	case appointmentmanager.BookingListViewActive:
		conditions = append(conditions, "b.status IN ('confirmed', 'queued', 'called', 'in_progress')")
	case appointmentmanager.BookingListViewCompleted:
		conditions = append(conditions, "b.status IN ('report_pending', 'completed', 'no_show')")
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM appointment_bookings b"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count bookings: %w", err)
	}
	queryArgs := append(args, filter.Limit, filter.Offset)
	rows, err := s.db.QueryContext(ctx, bookingSelect+where+" ORDER BY b.service_date DESC, b.created_at DESC, b.id DESC LIMIT ? OFFSET ?", queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()
	values := make([]appointmentmanager.Booking, 0)
	for rows.Next() {
		value, err := scanBooking(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan booking list: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate bookings: %w", err)
	}
	return values, total, nil
}

func (s *Store) WithinBookingTransaction(ctx context.Context, fn func(appointmentmanager.BookingTxStore) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin booking transaction: %w", err)
	}
	txStore := &bookingTxStore{tx: tx}
	if err := fn(txStore); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback booking transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit booking transaction: %w", err)
	}
	return nil
}

func (s *Store) WithinQueueTransaction(ctx context.Context, fn func(appointmentmanager.QueueTxStore) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin queue transaction: %w", err)
	}
	if err := fn(&bookingTxStore{tx: tx}); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback queue transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit queue transaction: %w", err)
	}
	return nil
}

type bookingScanner interface{ Scan(dest ...any) error }

func scanBooking(scanner bookingScanner) (appointmentmanager.Booking, error) {
	var value appointmentmanager.Booking
	var startedAt, examinationEndedAt, completedAt sql.NullTime
	var checkedInAt, calledAt, callDeadline sql.NullTime
	var building, roomNumber string
	var floorNumber int32
	err := scanner.Scan(
		&value.BookingID, &value.PatientAccountID, &value.PatientDisplayName,
		&value.PatientPhoneMasked, &value.PatientPhoneLast4,
		&value.DepartmentID, &value.ItemID, &value.ItemName,
		&value.RoomID, &value.CampusID, &building, &floorNumber, &roomNumber,
		&value.ServiceDate, &value.Session, &value.Status,
		&value.RoomOpenTime, &value.RoomCloseTime,
		&value.ItemStartTime, &value.ItemEndTime, &value.BookingCutoffTime,
		&value.EstimatedDurationMinutes,
		&startedAt, &value.StartedBy, &value.StartedByDisplayName,
		&examinationEndedAt, &value.ExaminationEndedBy, &value.ExaminationEndedByDisplayName,
		&completedAt, &value.CompletedBy, &value.CompletedByDisplayName,
		&value.QueueNumber, &value.CurrentCalledQueueNumber,
		&checkedInAt, &calledAt, &callDeadline, &value.CallAttempts, &value.PeopleAhead,
		&value.Version, &value.CreatedAt, &value.UpdatedAt,
	)
	if err != nil {
		return appointmentmanager.Booking{}, err
	}
	value.RoomDisplayName = appointmentmanager.FormatRoomDisplayName(building, floorNumber, roomNumber)
	value.Building, value.FloorNumber, value.RoomNumber = building, floorNumber, roomNumber
	if startedAt.Valid {
		value.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		value.CompletedAt = &completedAt.Time
	}
	if examinationEndedAt.Valid {
		value.ExaminationEndedAt = &examinationEndedAt.Time
	}
	if checkedInAt.Valid {
		value.CheckedInAt = &checkedInAt.Time
	}
	if calledAt.Valid {
		value.CalledAt = &calledAt.Time
	}
	if callDeadline.Valid {
		value.CallDeadline = &callDeadline.Time
	}
	return value, nil
}

var (
	_ patientmanager.BookingStore = (*Store)(nil)
	_ staffmanager.BookingStore   = (*Store)(nil)
)
