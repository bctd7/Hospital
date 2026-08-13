package staff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func currentBookingWeek() (time.Time, time.Time) {
	now := time.Now().In(bookingHospitalLocation)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, bookingHospitalLocation)
	weekday := (int(today.Weekday()) + 6) % 7
	weekStart := today.AddDate(0, 0, -weekday)
	return today, weekStart.AddDate(0, 0, 6)
}

func currentWeekDateForWeekday(weekday int32) (time.Time, bool) {
	today, weekEnd := currentBookingWeek()
	weekStart := weekEnd.AddDate(0, 0, -6)
	serviceDate := weekStart.AddDate(0, 0, int(weekday)-1)
	return serviceDate, !serviceDate.Before(today)
}

func deleteBookingsForConfiguration(
	ctx context.Context,
	tx BookingTxStore,
	operatorAccountID string,
	configurationOperationID string,
	filter BookingListFilter,
) ([]Booking, error) {
	bookings, err := tx.ListBookingsForUpdate(ctx, filter)
	if err != nil {
		return nil, err
	}
	for _, booking := range bookings {
		if booking.Status == BookingStatusCheckedIn {
			return nil, fmt.Errorf("%w: checked-in booking %s blocks configuration change", ErrInvalidState, booking.BookingID)
		}
	}
	deleted := make([]Booking, 0, len(bookings))
	for _, booking := range bookings {
		if booking.Status != BookingStatusConfirmed {
			continue
		}
		if err := deleteBookingForConfiguration(ctx, tx, operatorAccountID, configurationOperationID, booking); err != nil {
			return nil, err
		}
		deleted = append(deleted, booking)
	}
	return deleted, nil
}

func deleteBookingForConfiguration(ctx context.Context, tx BookingTxStore, operatorAccountID, configurationOperationID string, booking Booking) error {
	capacity, found, err := tx.FindDateCapacityForUpdate(ctx, booking.RoomID, booking.ServiceDate, booking.Session)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: capacity missing for booking %s", ErrInvalidState, booking.BookingID)
	}
	if err := tx.DecreaseOccupiedCapacity(ctx, capacity.CapacityID); err != nil {
		return err
	}
	if err := tx.ReleasePatientSession(ctx, booking.BookingID); err != nil {
		return err
	}
	if err := tx.DeleteBooking(ctx, booking.BookingID); err != nil {
		return err
	}
	fingerprint := sha256.Sum256([]byte(configurationOperationID + ":" + booking.BookingID))
	return tx.RecordBookingOperation(ctx, BookingOperationChange{
		OperationID: uuid.NewString(), OperatorAccountID: operatorAccountID,
		BookingID: booking.BookingID, Action: "delete_by_configuration",
		RequestFingerprint: hex.EncodeToString(fingerprint[:]),
		Result:             map[string]any{"booking_id": booking.BookingID, "deleted": true},
	})
}

func reconcileRoomWindowCapacity(
	ctx context.Context,
	tx BookingTxStore,
	operatorAccountID string,
	configurationOperationID string,
	window RoomWeeklyWindow,
) error {
	serviceDate, relevant := currentWeekDateForWeekday(window.Weekday)
	if !relevant {
		return nil
	}
	capacity, found, err := tx.FindDateCapacityForUpdate(ctx, window.RoomID, serviceDate, window.Session)
	if err != nil || !found {
		return err
	}
	bookings, err := tx.ListBookingsForUpdate(ctx, BookingListFilter{
		RoomID: window.RoomID, ServiceDate: &serviceDate, Session: window.Session,
	})
	if err != nil {
		return err
	}
	checkedIn := 0
	confirmed := make([]Booking, 0, len(bookings))
	for _, booking := range bookings {
		switch booking.Status {
		case BookingStatusCheckedIn:
			checkedIn++
		case BookingStatusConfirmed:
			confirmed = append(confirmed, booking)
		}
	}
	if int64(checkedIn) > window.ActiveCapacity {
		return fmt.Errorf("%w: checked-in bookings exceed requested capacity", ErrInvalidState)
	}
	excess := capacity.OccupiedCapacity - window.ActiveCapacity
	if excess > int64(len(confirmed)) {
		return fmt.Errorf("%w: checked-in bookings block capacity reduction", ErrInvalidState)
	}
	for index := int64(0); index < excess; index++ {
		if err := deleteBookingForConfiguration(ctx, tx, operatorAccountID, configurationOperationID, confirmed[index]); err != nil {
			return err
		}
	}
	return tx.UpdateDateCapacity(ctx, capacity.CapacityID, window.ActiveCapacity, window.WindowID, window.Version)
}
