package mysqlstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
)

const cleanupOperatorAccountID = "00000000-0000-0000-0000-000000000000"

type BookingCleanupResult struct {
	Processed     int
	MarkedNoShow  int
	Deferred      int
	DepartmentIDs []string
}

func (s *Store) CleanupExpiredBookings(ctx context.Context, before time.Time, limit int) (BookingCleanupResult, error) {
	if limit < 1 || limit > 500 {
		return BookingCleanupResult{}, fmt.Errorf("cleanup batch limit must be between 1 and 500")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return BookingCleanupResult{}, fmt.Errorf("begin booking cleanup: %w", err)
	}
	rollback := func(cause error) (BookingCleanupResult, error) {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, context.Canceled) {
			cause = errors.Join(cause, rollbackErr)
		}
		return BookingCleanupResult{}, cause
	}
	rows, err := tx.QueryContext(ctx, `
SELECT id
FROM appointment_bookings
WHERE (status = 'confirmed' AND (service_date < ? OR (service_date = ? AND item_cutoff_time_snapshot <= ?)))
   OR (status = 'called' AND id IN (
        SELECT booking_id FROM appointment_check_queue_entries WHERE call_deadline <= ?
   ))
ORDER BY service_date, item_cutoff_time_snapshot, id
LIMIT ? FOR UPDATE SKIP LOCKED`, before.Format("2006-01-02"), before.Format("2006-01-02"), before.Format("15:04:05"), before.UTC(), limit)
	if err != nil {
		return rollback(fmt.Errorf("lock expired bookings: %w", err))
	}
	bookingIDs := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return rollback(err)
		}
		bookingIDs = append(bookingIDs, id)
	}
	if err := rows.Close(); err != nil {
		return rollback(err)
	}
	if err := rows.Err(); err != nil {
		return rollback(err)
	}

	txStore := &bookingTxStore{tx: tx}
	departments := make(map[string]struct{})
	result := BookingCleanupResult{}
	for _, bookingID := range bookingIDs {
		booking, err := txStore.GetBookingForUpdate(ctx, bookingID)
		if err != nil {
			return rollback(err)
		}
		result.Processed++
		if booking.Status == appointmentmanager.BookingStatusCalled {
			entry, err := txStore.GetQueueEntryForUpdate(ctx, bookingID)
			if err != nil {
				return rollback(err)
			}
			queue, err := txStore.lockQueueByScope(ctx, booking.RoomID, booking.ServiceDate, booking.Session)
			if err != nil {
				return rollback(err)
			}
			end, err := cleanupDateTime(booking.ServiceDate, booking.ItemEndTime)
			if err != nil {
				return rollback(err)
			}
			booking.UpdatedAt = before
			if before.Before(end) {
				if err := txStore.MarkBookingDeferred(ctx, booking, queue, entry, uuid.NewString()); err != nil {
					return rollback(err)
				}
				result.Deferred++
				departments[booking.DepartmentID] = struct{}{}
				continue
			}
			if _, err := tx.ExecContext(ctx, `UPDATE appointment_check_queues SET current_called_booking_id = NULL, version = version + 1, updated_at = ? WHERE id = ? AND current_called_booking_id = ?`, before, queue.QueueID, bookingID); err != nil {
				return rollback(err)
			}
			if _, err := tx.ExecContext(ctx, `UPDATE appointment_check_queue_entries SET called_at = NULL, call_deadline = NULL, updated_at = ? WHERE booking_id = ?`, before, bookingID); err != nil {
				return rollback(err)
			}
		}
		capacity, found, err := txStore.FindDateCapacityForUpdate(ctx, booking.RoomID, booking.ServiceDate, booking.Session)
		if err != nil {
			return rollback(err)
		}
		if !found {
			return rollback(fmt.Errorf("%w: capacity missing for expired booking %s", appointmentmanager.ErrInvalidState, bookingID))
		}
		if err := txStore.DecreaseOccupiedCapacity(ctx, capacity.CapacityID); err != nil {
			return rollback(err)
		}
		if err := txStore.ReleasePatientItemSession(ctx, bookingID); err != nil {
			return rollback(err)
		}
		booking.Status, booking.UpdatedAt, booking.Version = appointmentmanager.BookingStatusNoShow, before, booking.Version+1
		if err := txStore.MarkBookingNoShow(ctx, booking, booking.Version-1); err != nil {
			return rollback(err)
		}
		if err := txStore.RecordBookingOperation(ctx, appointmentmanager.BookingOperationChange{OperationID: uuid.NewString(), OperatorAccountID: cleanupOperatorAccountID, BookingID: bookingID, Action: "mark_no_show", RequestFingerprint: "mark-no-show:" + bookingID, Result: booking}); err != nil {
			return rollback(err)
		}
		result.MarkedNoShow++
		departments[booking.DepartmentID] = struct{}{}
	}
	if err := tx.Commit(); err != nil {
		return BookingCleanupResult{}, fmt.Errorf("commit booking cleanup: %w", err)
	}
	result.DepartmentIDs = make([]string, 0, len(departments))
	for departmentID := range departments {
		result.DepartmentIDs = append(result.DepartmentIDs, departmentID)
	}
	return result, nil
}

func cleanupDateTime(date time.Time, clock string) (time.Time, error) {
	parsed, err := time.Parse("15:04:05", clock)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse cleanup time: %w", err)
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	date = date.In(location)
	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, location), nil
}
