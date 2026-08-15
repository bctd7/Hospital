package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
)

type bookingTxStore struct{ tx *sql.Tx }

func (s *bookingTxStore) ListBookingsForUpdate(ctx context.Context, filter appointmentmanager.BookingListFilter) ([]appointmentmanager.Booking, error) {
	conditions := make([]string, 0, 9)
	args := make([]any, 0, 9)
	add := func(condition string, value any) {
		conditions = append(conditions, condition)
		args = append(args, value)
	}
	if filter.PatientAccountID != "" {
		add("b.patient_account_id = ?", filter.PatientAccountID)
	}
	if filter.DepartmentID != "" {
		add("b.department_id = ?", filter.DepartmentID)
	}
	if filter.ServiceDate != nil {
		add("b.service_date = ?", filter.ServiceDate.Format("2006-01-02"))
	}
	if filter.FromDate != nil {
		add("b.service_date >= ?", filter.FromDate.Format("2006-01-02"))
	}
	if filter.ThroughDate != nil {
		add("b.service_date <= ?", filter.ThroughDate.Format("2006-01-02"))
	}
	if filter.Session != "" {
		add("b.session = ?", filter.Session)
	}
	if filter.ItemID != "" {
		add("b.item_id = ?", filter.ItemID)
	}
	if filter.RoomID != "" {
		add("b.room_id = ?", filter.RoomID)
	}
	if filter.Status != "" {
		add("b.status = ?", filter.Status)
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	rows, err := s.tx.QueryContext(ctx, bookingSelect+where+" ORDER BY b.created_at DESC, b.id DESC FOR UPDATE", args...)
	if err != nil {
		return nil, fmt.Errorf("lock bookings: %w", err)
	}
	defer rows.Close()
	values := make([]appointmentmanager.Booking, 0)
	for rows.Next() {
		value, err := scanBooking(rows)
		if err != nil {
			return nil, fmt.Errorf("scan locked booking: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locked bookings: %w", err)
	}
	return values, nil
}

func (s *bookingTxStore) FindBookingOperation(ctx context.Context, operationID string) (appointmentmanager.BookingOperation, bool, error) {
	var value appointmentmanager.BookingOperation
	err := s.tx.QueryRowContext(ctx, `
SELECT operator_account_id, booking_id, action, request_fingerprint, result_data
FROM appointment_booking_operations
WHERE operation_id = ?`, operationID).Scan(
		&value.OperatorAccountID, &value.BookingID, &value.Action, &value.RequestFingerprint, &value.ResultData,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.BookingOperation{}, false, nil
	}
	if err != nil {
		return appointmentmanager.BookingOperation{}, false, fmt.Errorf("find booking operation: %w", err)
	}
	return value, true, nil
}

func (s *bookingTxStore) ClaimPatientSession(ctx context.Context, patientAccountID string, serviceDate time.Time, session appointmentmanager.Session, bookingID string) error {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_patient_session_claims
    (patient_account_id, service_date, session, booking_id)
VALUES (?, ?, ?, ?)`, patientAccountID, serviceDate.Format("2006-01-02"), session, bookingID)
	if err != nil {
		if isDuplicateEntry(err) {
			return appointmentmanager.ErrPatientSessionOccupied
		}
		return fmt.Errorf("claim patient booking session: %w", err)
	}
	return nil
}

func (s *bookingTxStore) ReleasePatientSession(ctx context.Context, bookingID string) error {
	result, err := s.tx.ExecContext(ctx, `
DELETE FROM appointment_patient_session_claims
WHERE booking_id = ?`, bookingID)
	if err != nil {
		return fmt.Errorf("release patient booking session: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read patient session release result: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("%w: patient booking session claim is missing", appointmentmanager.ErrInvalidState)
	}
	return nil
}

func (s *bookingTxStore) ConsumePatientWeeklyQuota(ctx context.Context, patientAccountID string, weekStartDate time.Time, limit int64, now time.Time) error {
	if limit < 1 {
		return fmt.Errorf("%w: weekly quota limit must be positive", appointmentmanager.ErrInvalid)
	}
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_patient_weekly_quota_usage
    (patient_account_id, week_start_date, used_count, version, created_at, updated_at)
VALUES (?, ?, 0, 1, ?, ?)
ON DUPLICATE KEY UPDATE patient_account_id = patient_account_id`,
		patientAccountID, weekStartDate.Format("2006-01-02"), now, now,
	)
	if err != nil {
		return fmt.Errorf("ensure patient weekly quota: %w", err)
	}
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_patient_weekly_quota_usage
SET used_count = used_count + 1,
    version = version + 1,
    updated_at = ?
WHERE patient_account_id = ? AND week_start_date = ? AND used_count < ?`,
		now, patientAccountID, weekStartDate.Format("2006-01-02"), limit,
	)
	if err != nil {
		return fmt.Errorf("consume patient weekly quota: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read patient weekly quota result: %w", err)
	}
	if affected != 1 {
		return appointmentmanager.ErrPatientWeeklyQuotaFull
	}
	return nil
}

func (s *bookingTxStore) RecordBookingOperation(ctx context.Context, change appointmentmanager.BookingOperationChange) error {
	result, err := json.Marshal(change.Result)
	if err != nil {
		return fmt.Errorf("marshal booking operation result: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO appointment_booking_operations
    (operation_id, operator_account_id, booking_id, action, request_fingerprint, result_data)
VALUES (?, ?, ?, ?, ?, ?)`,
		change.OperationID, change.OperatorAccountID, change.BookingID,
		change.Action, change.RequestFingerprint, result,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: booking operation_id already exists", appointmentmanager.ErrConflict)
		}
		return fmt.Errorf("record booking operation: %w", err)
	}
	return nil
}

func (s *bookingTxStore) LockBookingSelection(ctx context.Context, itemID, roomID string, serviceDate time.Time, session appointmentmanager.Session) (appointmentmanager.BookingSelection, error) {
	var value appointmentmanager.BookingSelection
	var building, roomNumber string
	var floorNumber int32
	err := s.tx.QueryRowContext(ctx, `
SELECT i.id, i.owner_department_id, i.name, i.estimated_duration_minutes,
       r.id, r.campus_id, r.building, r.floor_number, r.room_number,
       iw.session, rw.id, rw.version,
       TIME_FORMAT(rw.open_time, '%H:%i:%s'), TIME_FORMAT(rw.close_time, '%H:%i:%s'), rw.active_capacity,
       TIME_FORMAT(iw.start_time, '%H:%i:%s'), TIME_FORMAT(iw.end_time, '%H:%i:%s'),
       TIME_FORMAT(iw.booking_cutoff_time, '%H:%i:%s')
FROM appointment_examination_items i
JOIN appointment_room_examination_items ri
  ON ri.item_id = i.id AND ri.room_id = ? AND ri.status = 'active'
JOIN appointment_rooms r
  ON r.id = ri.room_id AND r.department_id = i.owner_department_id AND r.retired_at IS NULL
JOIN appointment_item_weekly_windows iw
  ON iw.item_id = i.id
 AND iw.weekday = WEEKDAY(?) + 1
 AND iw.session = ?
 AND iw.status = 'active'
JOIN appointment_room_weekly_windows rw
  ON rw.room_id = r.id
 AND rw.weekday = iw.weekday
 AND rw.session = iw.session
 AND rw.status = 'active'
WHERE i.id = ?
  AND i.status = 'active'
  AND rw.open_time <= iw.start_time
  AND iw.end_time <= rw.close_time
FOR UPDATE`, roomID, serviceDate.Format("2006-01-02"), session, itemID).Scan(
		&value.ItemID, &value.DepartmentID, &value.ItemName, &value.EstimatedDurationMinutes,
		&value.RoomID, &value.CampusID, &building, &floorNumber, &roomNumber,
		&value.Session, &value.RoomWindowID, &value.RoomWindowVersion,
		&value.RoomOpenTime, &value.RoomCloseTime, &value.ActiveCapacity,
		&value.ItemStartTime, &value.ItemEndTime, &value.BookingCutoffTime,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.BookingSelection{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.BookingSelection{}, fmt.Errorf("lock booking selection: %w", err)
	}
	value.RoomDisplayName = appointmentmanager.FormatRoomDisplayName(building, floorNumber, roomNumber)
	value.Building, value.FloorNumber, value.RoomNumber = building, floorNumber, roomNumber
	return value, nil
}

func (s *bookingTxStore) FindDateCapacityForUpdate(ctx context.Context, roomID string, serviceDate time.Time, session appointmentmanager.Session) (appointmentmanager.DateCapacity, bool, error) {
	var value appointmentmanager.DateCapacity
	err := s.tx.QueryRowContext(ctx, `
SELECT id, room_id, service_date, session, total_capacity, occupied_capacity,
       room_window_id, room_window_version, version, created_at, updated_at
FROM appointment_room_date_capacity
WHERE room_id = ? AND service_date = ? AND session = ?
FOR UPDATE`, roomID, serviceDate.Format("2006-01-02"), session).Scan(
		&value.CapacityID, &value.RoomID, &value.ServiceDate, &value.Session,
		&value.TotalCapacity, &value.OccupiedCapacity, &value.RoomWindowID,
		&value.RoomWindowVersion, &value.Version, &value.CreatedAt, &value.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.DateCapacity{}, false, nil
	}
	if err != nil {
		return appointmentmanager.DateCapacity{}, false, fmt.Errorf("lock date capacity: %w", err)
	}
	return value, true, nil
}

func (s *bookingTxStore) CreateDateCapacity(ctx context.Context, value appointmentmanager.DateCapacity) error {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_room_date_capacity
    (id, room_id, service_date, session, total_capacity, occupied_capacity,
     room_window_id, room_window_version, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE id = id`,
		value.CapacityID, value.RoomID, value.ServiceDate.Format("2006-01-02"), value.Session,
		value.TotalCapacity, value.OccupiedCapacity, value.RoomWindowID,
		value.RoomWindowVersion, value.Version, value.CreatedAt, value.UpdatedAt,
	)
	return mapWriteError(err, "create date capacity")
}

func (s *bookingTxStore) IncreaseOccupiedCapacity(ctx context.Context, capacityID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_room_date_capacity
SET occupied_capacity = occupied_capacity + 1,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND occupied_capacity < total_capacity`, capacityID)
	if err != nil {
		return fmt.Errorf("increase occupied capacity: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read occupied capacity result: %w", err)
	}
	if affected != 1 {
		return appointmentmanager.ErrCapacityFull
	}
	return nil
}

func (s *bookingTxStore) DecreaseOccupiedCapacity(ctx context.Context, capacityID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_room_date_capacity
SET occupied_capacity = occupied_capacity - 1,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND occupied_capacity > 0`, capacityID)
	if err != nil {
		return fmt.Errorf("decrease occupied capacity: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read released capacity result: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("%w: booking capacity was already released", appointmentmanager.ErrInvalidState)
	}
	return nil
}

func (s *bookingTxStore) UpdateDateCapacity(ctx context.Context, capacityID string, totalCapacity int64, roomWindowID string, roomWindowVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_room_date_capacity
SET total_capacity = ?, room_window_id = ?, room_window_version = ?,
    version = version + 1, updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND occupied_capacity <= ?`, totalCapacity, roomWindowID, roomWindowVersion, capacityID, totalCapacity)
	if err != nil {
		return fmt.Errorf("update date capacity: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read date capacity update result: %w", err)
	}
	if affected != 1 {
		return appointmentmanager.ErrInvalidState
	}
	return nil
}

func (s *bookingTxStore) GetBookingForUpdate(ctx context.Context, bookingID string) (appointmentmanager.Booking, error) {
	value, err := scanBooking(s.tx.QueryRowContext(ctx, bookingSelect+" WHERE b.id = ? FOR UPDATE", bookingID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.Booking{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.Booking{}, fmt.Errorf("lock booking: %w", err)
	}
	return value, nil
}

func (s *bookingTxStore) CreateBooking(ctx context.Context, value appointmentmanager.Booking) error {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_bookings
    (id, patient_account_id, patient_display_name_snapshot, patient_phone_masked_snapshot,
     patient_phone_last4_snapshot, department_id, item_id, room_id, service_date, session, status,
     room_open_time_snapshot, room_close_time_snapshot, item_start_time_snapshot,
     item_end_time_snapshot, item_cutoff_time_snapshot,
     estimated_duration_minutes_snapshot, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.BookingID, value.PatientAccountID, value.PatientDisplayName, value.PatientPhoneMasked,
		value.PatientPhoneLast4, value.DepartmentID, value.ItemID, value.RoomID,
		value.ServiceDate.Format("2006-01-02"), value.Session, value.Status,
		value.RoomOpenTime, value.RoomCloseTime, value.ItemStartTime,
		value.ItemEndTime, value.BookingCutoffTime, value.EstimatedDurationMinutes,
		value.Version, value.CreatedAt, value.UpdatedAt,
	)
	return mapWriteError(err, "create booking")
}

func (s *bookingTxStore) StartExamination(ctx context.Context, value appointmentmanager.Booking, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_bookings
SET status = 'in_progress', started_at = ?, started_by = ?, started_by_display_name_snapshot = ?,
    version = version + 1, updated_at = ?
WHERE id = ? AND status = 'confirmed' AND version = ?`,
		value.StartedAt, value.StartedBy, value.StartedByDisplayName, value.UpdatedAt, value.BookingID, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("start examination: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read start examination result: %w", err)
	}
	if affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	return nil
}

func (s *bookingTxStore) MarkBookingNoShow(ctx context.Context, value appointmentmanager.Booking, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_bookings
SET status = 'no_show', version = version + 1, updated_at = ?
WHERE id = ? AND status = 'confirmed' AND version = ?`,
		value.UpdatedAt, value.BookingID, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("mark booking no-show: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read no-show result: %w", err)
	}
	if affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	return nil
}

func (s *bookingTxStore) CompleteBooking(ctx context.Context, value appointmentmanager.Booking, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_bookings
SET status = 'completed', completed_at = ?, completed_by = ?, completed_by_display_name_snapshot = ?,
    version = version + 1, updated_at = ?
WHERE id = ? AND status = 'in_progress' AND version = ?`,
		value.CompletedAt, value.CompletedBy, value.CompletedByDisplayName, value.UpdatedAt, value.BookingID, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("complete examination: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read complete examination result: %w", err)
	}
	if affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	return nil
}

func (s *bookingTxStore) DeleteBooking(ctx context.Context, bookingID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_bookings
SET status = 'canceled', version = version + 1, updated_at = ?
WHERE id = ? AND status = 'confirmed'`, time.Now().UTC(), bookingID)
	if err != nil {
		return fmt.Errorf("cancel booking: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read cancel booking result: %w", err)
	}
	if affected != 1 {
		return appointmentmanager.ErrNotFound
	}
	return nil
}

var _ appointmentmanager.BookingTxStore = (*bookingTxStore)(nil)
