package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
)

func (s *bookingTxStore) CheckInBooking(ctx context.Context, booking appointmentmanager.Booking, expectedVersion int64, queueID, eventID string) (appointmentmanager.QueueEntry, error) {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_check_queues
    (id, room_id, service_date, session, next_ticket_number, call_sequence, version, created_at, updated_at)
VALUES (?, ?, ?, ?, 1, 0, 1, ?, ?)
ON DUPLICATE KEY UPDATE id = id`, queueID, booking.RoomID, booking.ServiceDate.Format("2006-01-02"), booking.Session, booking.UpdatedAt, booking.UpdatedAt)
	if err != nil {
		return appointmentmanager.QueueEntry{}, fmt.Errorf("ensure check queue: %w", err)
	}
	queue, err := s.lockQueueByScope(ctx, booking.RoomID, booking.ServiceDate, booking.Session)
	if err != nil {
		return appointmentmanager.QueueEntry{}, err
	}
	entry := appointmentmanager.QueueEntry{
		BookingID: booking.BookingID, QueueID: queue.QueueID, TicketNumber: queue.NextTicketNumber,
		CheckedInAt: booking.UpdatedAt,
	}
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_bookings
SET status = 'queued', version = version + 1, updated_at = ?
WHERE id = ? AND status = 'confirmed' AND version = ?`, booking.UpdatedAt, booking.BookingID, expectedVersion)
	if err != nil {
		return appointmentmanager.QueueEntry{}, fmt.Errorf("mark booking queued: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.QueueEntry{}, appointmentmanager.ErrVersionConflict
	}
	if _, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_check_queue_entries
    (booking_id, queue_id, ticket_number, eligible_after_call_sequence, call_attempts, checked_in_at, updated_at)
VALUES (?, ?, ?, 0, 0, ?, ?)`, booking.BookingID, queue.QueueID, entry.TicketNumber, entry.CheckedInAt, booking.UpdatedAt); err != nil {
		return appointmentmanager.QueueEntry{}, mapWriteError(err, "create check queue entry")
	}
	result, err = s.tx.ExecContext(ctx, `
UPDATE appointment_check_queues
SET next_ticket_number = next_ticket_number + 1, version = version + 1, updated_at = ?
WHERE id = ? AND version = ?`, booking.UpdatedAt, queue.QueueID, queue.Version)
	if err != nil {
		return appointmentmanager.QueueEntry{}, fmt.Errorf("advance check queue ticket: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.QueueEntry{}, appointmentmanager.ErrVersionConflict
	}
	if err := s.RecordQueueEvent(ctx, appointmentmanager.QueueEvent{EventID: eventID, QueueID: queue.QueueID, BookingID: booking.BookingID, EventType: appointmentmanager.QueueEventCheckedIn, OccurredAt: booking.UpdatedAt}); err != nil {
		return appointmentmanager.QueueEntry{}, err
	}
	return entry, nil
}

func (s *bookingTxStore) lockQueueByScope(ctx context.Context, roomID string, serviceDate time.Time, session appointmentmanager.Session) (appointmentmanager.CheckQueue, error) {
	var value appointmentmanager.CheckQueue
	var current sql.NullString
	err := s.tx.QueryRowContext(ctx, `
SELECT id, room_id, service_date, session, next_ticket_number, call_sequence,
       current_called_booking_id, version
FROM appointment_check_queues
WHERE room_id = ? AND service_date = ? AND session = ?
FOR UPDATE`, roomID, serviceDate.Format("2006-01-02"), session).Scan(
		&value.QueueID, &value.RoomID, &value.ServiceDate, &value.Session, &value.NextTicketNumber,
		&value.CallSequence, &current, &value.Version,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.CheckQueue{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.CheckQueue{}, fmt.Errorf("lock check queue: %w", err)
	}
	if current.Valid {
		value.CurrentCalledBookingID = current.String
	}
	return value, nil
}

func (s *bookingTxStore) ListRoomQueuesForUpdate(ctx context.Context, roomID string, serviceDate time.Time) ([]appointmentmanager.CheckQueue, error) {
	rows, err := s.tx.QueryContext(ctx, `
SELECT id, room_id, service_date, session, next_ticket_number, call_sequence,
       COALESCE(current_called_booking_id, ''), version
FROM appointment_check_queues
WHERE room_id = ? AND service_date = ?
ORDER BY CASE session WHEN 'morning' THEN 0 ELSE 1 END, id
FOR UPDATE`, roomID, serviceDate.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("lock room check queues: %w", err)
	}
	defer rows.Close()
	values := make([]appointmentmanager.CheckQueue, 0, 2)
	for rows.Next() {
		var value appointmentmanager.CheckQueue
		if err := rows.Scan(&value.QueueID, &value.RoomID, &value.ServiceDate, &value.Session, &value.NextTicketNumber, &value.CallSequence, &value.CurrentCalledBookingID, &value.Version); err != nil {
			return nil, fmt.Errorf("scan room check queue: %w", err)
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *bookingTxStore) HasInProgressBookingForUpdate(ctx context.Context, roomID string, serviceDate time.Time) (bool, error) {
	var bookingID string
	err := s.tx.QueryRowContext(ctx, `
SELECT id FROM appointment_bookings
WHERE room_id = ? AND service_date = ? AND status = 'in_progress'
ORDER BY started_at, id LIMIT 1 FOR UPDATE`, roomID, serviceDate.Format("2006-01-02")).Scan(&bookingID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lock in-progress room booking: %w", err)
	}
	return true, nil
}

func (s *bookingTxStore) FindNextQueueBookingForUpdate(ctx context.Context, queue appointmentmanager.CheckQueue, now time.Time) (appointmentmanager.Booking, appointmentmanager.QueueEntry, bool, error) {
	bookingID, found, err := s.findQueueCandidate(ctx, queue.QueueID, queue.CallSequence, now, true)
	if err != nil {
		return appointmentmanager.Booking{}, appointmentmanager.QueueEntry{}, false, err
	}
	if !found {
		bookingID, found, err = s.findQueueCandidate(ctx, queue.QueueID, queue.CallSequence, now, false)
		if err != nil || !found {
			return appointmentmanager.Booking{}, appointmentmanager.QueueEntry{}, false, err
		}
	}
	booking, err := s.GetBookingForUpdate(ctx, bookingID)
	if err != nil {
		return appointmentmanager.Booking{}, appointmentmanager.QueueEntry{}, false, err
	}
	entry, err := s.lockQueueEntry(ctx, bookingID)
	if err != nil {
		return appointmentmanager.Booking{}, appointmentmanager.QueueEntry{}, false, err
	}
	return booking, entry, true, nil
}

func (s *bookingTxStore) findQueueCandidate(ctx context.Context, queueID string, callSequence int64, now time.Time, requireEligible bool) (string, bool, error) {
	condition := ""
	if requireEligible {
		condition = " AND e.eligible_after_call_sequence <= ?"
	}
	// 预约日期和窗口时间是医院本地墙上时间；显式传字符串，避免 DSN 的 UTC loc
	// 把东八区 time.Time 换算到前一天，尤其会破坏周日当天的叫号判断。
	args := []any{queueID, now.Format("2006-01-02 15:04:05.000")}
	if requireEligible {
		args = append(args, callSequence)
	}
	var bookingID string
	err := s.tx.QueryRowContext(ctx, `
SELECT e.booking_id
FROM appointment_check_queue_entries e
JOIN appointment_bookings b ON b.id = e.booking_id
WHERE e.queue_id = ? AND b.status = 'queued'
  AND TIMESTAMP(b.service_date, b.item_start_time_snapshot) <= ?`+condition+`
ORDER BY e.ticket_number, e.booking_id
LIMIT 1 FOR UPDATE`, args...).Scan(&bookingID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("lock next queue candidate: %w", err)
	}
	return bookingID, true, nil
}

func (s *bookingTxStore) lockQueueEntry(ctx context.Context, bookingID string) (appointmentmanager.QueueEntry, error) {
	var value appointmentmanager.QueueEntry
	var calledAt, deadline sql.NullTime
	err := s.tx.QueryRowContext(ctx, `
SELECT booking_id, queue_id, ticket_number, eligible_after_call_sequence, call_attempts,
       checked_in_at, called_at, call_deadline
FROM appointment_check_queue_entries WHERE booking_id = ? FOR UPDATE`, bookingID).Scan(
		&value.BookingID, &value.QueueID, &value.TicketNumber, &value.EligibleAfterCallSequence,
		&value.CallAttempts, &value.CheckedInAt, &calledAt, &deadline,
	)
	if err != nil {
		return appointmentmanager.QueueEntry{}, fmt.Errorf("lock check queue entry: %w", err)
	}
	if calledAt.Valid {
		value.CalledAt = &calledAt.Time
	}
	if deadline.Valid {
		value.CallDeadline = &deadline.Time
	}
	return value, nil
}

func (s *bookingTxStore) GetQueueEntryForUpdate(ctx context.Context, bookingID string) (appointmentmanager.QueueEntry, error) {
	return s.lockQueueEntry(ctx, bookingID)
}

func (s *bookingTxStore) MarkBookingCalled(ctx context.Context, booking appointmentmanager.Booking, queue appointmentmanager.CheckQueue, entry appointmentmanager.QueueEntry, expectedVersion int64, eventID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_check_queues
SET call_sequence = call_sequence + 1, current_called_booking_id = ?, version = version + 1, updated_at = ?
WHERE id = ? AND current_called_booking_id IS NULL AND version = ?`, booking.BookingID, booking.UpdatedAt, queue.QueueID, queue.Version)
	if err != nil {
		return fmt.Errorf("claim current called booking: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrInvalidState
	}
	result, err = s.tx.ExecContext(ctx, `
UPDATE appointment_bookings SET status = 'called', version = version + 1, updated_at = ?
WHERE id = ? AND status = 'queued' AND version = ?`, booking.UpdatedAt, booking.BookingID, expectedVersion)
	if err != nil {
		return fmt.Errorf("mark booking called: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	result, err = s.tx.ExecContext(ctx, `
UPDATE appointment_check_queue_entries
SET call_attempts = call_attempts + 1, called_at = ?, call_deadline = ?, updated_at = ?
WHERE booking_id = ?`, entry.CalledAt, entry.CallDeadline, booking.UpdatedAt, booking.BookingID)
	if err != nil {
		return fmt.Errorf("update called queue entry: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrInvalidState
	}
	return s.RecordQueueEvent(ctx, appointmentmanager.QueueEvent{EventID: eventID, QueueID: queue.QueueID, BookingID: booking.BookingID, EventType: appointmentmanager.QueueEventCalled, CallSequence: queue.CallSequence + 1, OccurredAt: booking.UpdatedAt})
}

func (s *bookingTxStore) MarkBookingDeferred(ctx context.Context, booking appointmentmanager.Booking, queue appointmentmanager.CheckQueue, entry appointmentmanager.QueueEntry, eventID string) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_check_queues SET current_called_booking_id = NULL, version = version + 1, updated_at = ? WHERE id = ? AND current_called_booking_id = ?`, booking.UpdatedAt, queue.QueueID, booking.BookingID)
	if err != nil {
		return fmt.Errorf("clear called booking: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrInvalidState
	}
	result, err = s.tx.ExecContext(ctx, `UPDATE appointment_bookings SET status = 'queued', version = version + 1, updated_at = ? WHERE id = ? AND status = 'called'`, booking.UpdatedAt, booking.BookingID)
	if err != nil {
		return fmt.Errorf("defer called booking: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrInvalidState
	}
	_, err = s.tx.ExecContext(ctx, `UPDATE appointment_check_queue_entries SET eligible_after_call_sequence = ?, called_at = NULL, call_deadline = NULL, updated_at = ? WHERE booking_id = ?`, queue.CallSequence+3, booking.UpdatedAt, booking.BookingID)
	if err != nil {
		return fmt.Errorf("update deferred queue entry: %w", err)
	}
	return s.RecordQueueEvent(ctx, appointmentmanager.QueueEvent{EventID: eventID, QueueID: queue.QueueID, BookingID: booking.BookingID, EventType: appointmentmanager.QueueEventDeferred, CallSequence: queue.CallSequence, OccurredAt: booking.UpdatedAt})
}

func (s *bookingTxStore) RecordQueueEvent(ctx context.Context, event appointmentmanager.QueueEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.NewString()
	}
	_, err := s.tx.ExecContext(ctx, `INSERT INTO appointment_check_queue_events (id, queue_id, booking_id, event_type, call_sequence, occurred_at) VALUES (?, ?, ?, ?, ?, ?)`, event.EventID, event.QueueID, event.BookingID, event.EventType, event.CallSequence, event.OccurredAt)
	return mapWriteError(err, "record check queue event")
}
