package patient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/appointment/rpc/internal/manager/common"
)

const bookingActionCheckIn = "check_in"

type CheckInCommand struct {
	BookingID       string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

// CheckIn 把本人待到场预约加入其房间、日期和大窗的候检队列。
func (m *Manager) CheckIn(ctx context.Context, patient authn.Principal, command CheckInCommand) (Booking, error) {
	if err := requirePatient(patient); err != nil {
		return Booking{}, err
	}
	var err error
	command.BookingID, err = normalizeUUID(command.BookingID, "booking_id")
	if err != nil {
		return Booking{}, err
	}
	command.OperationID, err = normalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return Booking{}, err
	}
	if command.ExpectedVersion < 1 {
		return Booking{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	fingerprint := common.RequestFingerprint(struct {
		BookingID       string
		ExpectedVersion int64
	}{command.BookingID, command.ExpectedVersion})
	var result Booking
	err = m.bookings.WithinQueueTransaction(ctx, func(tx QueueTxStore) error {
		operation, found, findErr := tx.FindBookingOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if operation.OperatorAccountID != patient.AccountID || operation.BookingID != command.BookingID || operation.Action != bookingActionCheckIn || operation.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			return json.Unmarshal(operation.ResultData, &result)
		}
		booking, lockErr := tx.GetBookingForUpdate(ctx, command.BookingID)
		if lockErr != nil {
			return lockErr
		}
		if booking.PatientAccountID != patient.AccountID {
			return ErrNotFound
		}
		if booking.Status != BookingStatusConfirmed {
			return ErrInvalidState
		}
		now := time.Now().In(hospitalLocation)
		start, parseErr := dateTime(booking.ServiceDate, booking.ItemStartTime)
		if parseErr != nil {
			return parseErr
		}
		cutoff, parseErr := dateTime(booking.ServiceDate, booking.BookingCutoffTime)
		if parseErr != nil {
			return parseErr
		}
		if now.Before(start) || !now.Before(cutoff) {
			return ErrBookingClosed
		}
		booking.Status = BookingStatusQueued
		booking.UpdatedAt = now
		booking.Version = command.ExpectedVersion + 1
		entry, joinErr := tx.CheckInBooking(ctx, booking, command.ExpectedVersion, uuid.NewString(), uuid.NewString())
		if joinErr != nil {
			return joinErr
		}
		booking.QueueNumber = entry.TicketNumber
		checkedInAt := entry.CheckedInAt
		booking.CheckedInAt = &checkedInAt
		result = booking
		return tx.RecordBookingOperation(ctx, BookingOperationChange{OperationID: command.OperationID, OperatorAccountID: patient.AccountID, BookingID: booking.BookingID, Action: bookingActionCheckIn, RequestFingerprint: fingerprint, Result: result})
	})
	if err != nil {
		return Booking{}, err
	}
	m.invalidateBookingReads(ctx, result.DepartmentID)
	return result, nil
}
