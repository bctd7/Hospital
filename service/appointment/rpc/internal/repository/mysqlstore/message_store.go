package mysqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
)

func (s *Store) ListMessageBookings(ctx context.Context, patientAccountID, departmentID string) ([]appointmentmanager.Booking, error) {
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if patientAccountID != "" {
		conditions, args = append(conditions, "b.patient_account_id = ?"), append(args, patientAccountID)
	}
	if departmentID != "" {
		conditions, args = append(conditions, "b.department_id = ?"), append(args, departmentID)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	rows, err := s.db.QueryContext(ctx, bookingSelect+where+" ORDER BY b.created_at DESC, b.id DESC", args...)
	if err != nil {
		return nil, fmt.Errorf("list message bookings: %w", err)
	}
	defer rows.Close()
	values := make([]appointmentmanager.Booking, 0)
	for rows.Next() {
		value, scanErr := scanBooking(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan message booking: %w", scanErr)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate message bookings: %w", err)
	}
	return values, nil
}

func (s *Store) ListPatientMessageReportFacts(ctx context.Context, patientAccountID string) ([]appointmentmanager.MessageReportFact, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT r.id, v.id, v.version_no, v.version_kind, v.published_at,
       b.id, b.patient_account_id, b.patient_display_name_snapshot,
       b.patient_phone_masked_snapshot, b.patient_phone_last4_snapshot,
       b.department_id, b.item_id, i.name,
       b.room_id, room.campus_id, room.building, room.floor_number, room.room_number,
       b.service_date, b.session, b.status,
       TIME_FORMAT(b.room_open_time_snapshot, '%H:%i:%s'),
       TIME_FORMAT(b.room_close_time_snapshot, '%H:%i:%s'),
       TIME_FORMAT(b.item_start_time_snapshot, '%H:%i:%s'),
       TIME_FORMAT(b.item_end_time_snapshot, '%H:%i:%s'),
       TIME_FORMAT(b.item_cutoff_time_snapshot, '%H:%i:%s'),
       b.started_at, COALESCE(b.started_by, ''), COALESCE(b.started_by_display_name_snapshot, ''),
       b.completed_at, COALESCE(b.completed_by, ''), COALESCE(b.completed_by_display_name_snapshot, ''),
       b.version, b.created_at, b.updated_at
FROM appointment_examination_reports r
JOIN appointment_examination_report_versions v ON v.report_id = r.id
JOIN appointment_bookings b ON b.id = r.booking_id
JOIN appointment_examination_items i ON i.id = b.item_id
JOIN appointment_rooms room ON room.id = b.room_id
WHERE r.patient_account_id = ? AND v.published_at IS NOT NULL
  AND v.version_kind IN ('initial', 'correction')
ORDER BY v.published_at DESC, v.id DESC`, patientAccountID)
	if err != nil {
		return nil, fmt.Errorf("list patient message report facts: %w", err)
	}
	defer rows.Close()
	values := make([]appointmentmanager.MessageReportFact, 0)
	for rows.Next() {
		var value appointmentmanager.MessageReportFact
		var publishedAt time.Time
		booking, scanErr := scanMessageReportBooking(rows, &value, &publishedAt)
		if scanErr != nil {
			return nil, fmt.Errorf("scan patient message report fact: %w", scanErr)
		}
		value.Booking, value.PublishedAt = booking, publishedAt
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patient message report facts: %w", err)
	}
	return values, nil
}

func (s *Store) ListPatientMessageQueueFacts(ctx context.Context, patientAccountID string) ([]appointmentmanager.MessageQueueFact, error) {
	bookings, err := s.ListMessageBookings(ctx, patientAccountID, "")
	if err != nil {
		return nil, err
	}
	byID := make(map[string]appointmentmanager.Booking, len(bookings))
	for _, booking := range bookings {
		byID[booking.BookingID] = booking
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT e.id, e.event_type, e.call_sequence, e.occurred_at, e.booking_id
FROM appointment_check_queue_events e
JOIN appointment_bookings b ON b.id = e.booking_id
WHERE b.patient_account_id = ? AND e.event_type IN ('called', 'deferred')
ORDER BY e.occurred_at DESC, e.id DESC`, patientAccountID)
	if err != nil {
		return nil, fmt.Errorf("list patient queue message facts: %w", err)
	}
	defer rows.Close()
	values := make([]appointmentmanager.MessageQueueFact, 0)
	for rows.Next() {
		var value appointmentmanager.MessageQueueFact
		var bookingID string
		if err := rows.Scan(&value.EventID, &value.EventType, &value.CallSequence, &value.OccurredAt, &bookingID); err != nil {
			return nil, fmt.Errorf("scan patient queue message fact: %w", err)
		}
		value.Booking = byID[bookingID]
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate patient queue message facts: %w", err)
	}
	return values, nil
}

func scanMessageReportBooking(scanner bookingScanner, fact *appointmentmanager.MessageReportFact, publishedAt *time.Time) (appointmentmanager.Booking, error) {
	var value appointmentmanager.Booking
	var startedAt, completedAt sql.NullTime
	var building, roomNumber string
	var floorNumber int32
	err := scanner.Scan(
		&fact.ReportID, &fact.VersionID, &fact.VersionNo, &fact.VersionKind, publishedAt,
		&value.BookingID, &value.PatientAccountID, &value.PatientDisplayName,
		&value.PatientPhoneMasked, &value.PatientPhoneLast4,
		&value.DepartmentID, &value.ItemID, &value.ItemName,
		&value.RoomID, &value.CampusID, &building, &floorNumber, &roomNumber,
		&value.ServiceDate, &value.Session, &value.Status,
		&value.RoomOpenTime, &value.RoomCloseTime, &value.ItemStartTime, &value.ItemEndTime, &value.BookingCutoffTime,
		&startedAt, &value.StartedBy, &value.StartedByDisplayName,
		&completedAt, &value.CompletedBy, &value.CompletedByDisplayName,
		&value.Version, &value.CreatedAt, &value.UpdatedAt,
	)
	if err != nil {
		return appointmentmanager.Booking{}, err
	}
	value.Building, value.FloorNumber, value.RoomNumber = building, floorNumber, roomNumber
	value.RoomDisplayName = appointmentmanager.FormatRoomDisplayName(building, floorNumber, roomNumber)
	if startedAt.Valid {
		value.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		value.CompletedAt = &completedAt.Time
	}
	return value, nil
}

func (s *Store) ListMessageReads(ctx context.Context, accountID string, messageKeys []string) (map[string]time.Time, error) {
	values := make(map[string]time.Time, len(messageKeys))
	if len(messageKeys) == 0 {
		return values, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(messageKeys)), ",")
	args := make([]any, 0, len(messageKeys)+1)
	args = append(args, accountID)
	for _, key := range messageKeys {
		args = append(args, key)
	}
	rows, err := s.db.QueryContext(ctx, "SELECT message_key, read_at FROM appointment_message_reads WHERE account_id = ? AND message_key IN ("+placeholders+")", args...)
	if err != nil {
		return nil, fmt.Errorf("list message reads: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var readAt time.Time
		if err := rows.Scan(&key, &readAt); err != nil {
			return nil, fmt.Errorf("scan message read: %w", err)
		}
		values[key] = readAt
	}
	return values, rows.Err()
}

func (s *Store) MarkMessageRead(ctx context.Context, accountID, messageKey string, readAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO appointment_message_reads (account_id, message_key, read_at)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE read_at = VALUES(read_at)`, accountID, messageKey, readAt)
	if err != nil {
		return fmt.Errorf("mark message read: %w", err)
	}
	return nil
}

var _ appointmentmanager.MessageStore = (*Store)(nil)
