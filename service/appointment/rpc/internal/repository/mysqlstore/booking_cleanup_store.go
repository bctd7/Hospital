package mysqlstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
)

const cleanupOperatorAccountID = "00000000-0000-0000-0000-000000000000"

type BookingCleanupResult struct {
	MarkedNoShow  int
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
WHERE status = 'confirmed'
  AND (service_date < ? OR (service_date = ? AND item_end_time_snapshot <= ?))
ORDER BY service_date, item_end_time_snapshot, id
LIMIT ?
FOR UPDATE SKIP LOCKED`, before.Format("2006-01-02"), before.Format("2006-01-02"), before.Format("15:04:05"), limit)
	if err != nil {
		return rollback(fmt.Errorf("lock expired bookings: %w", err))
	}
	bookingIDs := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return rollback(fmt.Errorf("scan expired booking: %w", err))
		}
		bookingIDs = append(bookingIDs, id)
	}
	if err := rows.Close(); err != nil {
		return rollback(fmt.Errorf("close expired booking rows: %w", err))
	}
	if err := rows.Err(); err != nil {
		return rollback(fmt.Errorf("iterate expired bookings: %w", err))
	}

	txStore := &bookingTxStore{tx: tx}
	departments := make(map[string]struct{})
	for _, bookingID := range bookingIDs {
		booking, err := txStore.GetBookingForUpdate(ctx, bookingID)
		if err != nil {
			return rollback(err)
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
		booking.Status = appointmentmanager.BookingStatusNoShow
		booking.UpdatedAt = before
		booking.Version++
		if err := txStore.MarkBookingNoShow(ctx, booking, booking.Version-1); err != nil {
			return rollback(err)
		}
		fingerprint := sha256.Sum256([]byte("mark_no_show:" + bookingID))
		if err := txStore.RecordBookingOperation(ctx, appointmentmanager.BookingOperationChange{
			OperationID: uuid.NewString(), OperatorAccountID: cleanupOperatorAccountID,
			BookingID: bookingID, Action: "mark_no_show",
			RequestFingerprint: hex.EncodeToString(fingerprint[:]),
			Result:             booking,
		}); err != nil {
			return rollback(err)
		}
		departments[booking.DepartmentID] = struct{}{}
	}
	if err := tx.Commit(); err != nil {
		return BookingCleanupResult{}, fmt.Errorf("commit booking cleanup: %w", err)
	}
	result := BookingCleanupResult{MarkedNoShow: len(bookingIDs), DepartmentIDs: make([]string, 0, len(departments))}
	for departmentID := range departments {
		result.DepartmentIDs = append(result.DepartmentIDs, departmentID)
	}
	return result, nil
}
