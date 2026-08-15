package staff

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	"hospital/service/appointment/rpc/internal/manager/common"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

const bookingActionCallNext = "call_next"

// CallNext 按房间当天较早大窗、报到顺序和顺延轮次选择唯一下一位患者。
func (m *Manager) CallNext(ctx context.Context, operator authn.Principal, command staffinput.CallNext) (Booking, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return Booking{}, err
	}
	var err error
	command.DepartmentID, err = staffsupport.ScopedDepartment(operator, command.DepartmentID)
	if err != nil {
		return Booking{}, err
	}
	command.RoomID, err = staffsupport.NormalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return Booking{}, err
	}
	command.OperationID, err = staffsupport.NormalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return Booking{}, err
	}
	serviceDate, err := time.ParseInLocation(bookingDateLayout, command.ServiceDate, bookingHospitalLocation)
	if err != nil {
		return Booking{}, fmt.Errorf("%w: service_date must use YYYY-MM-DD", ErrInvalid)
	}
	now := time.Now().In(bookingHospitalLocation)
	if !sameBookingDate(serviceDate, now) {
		return Booking{}, fmt.Errorf("%w: queue operations are only allowed today", ErrInvalidState)
	}
	room, err := m.rooms.GetRoom(ctx, command.RoomID)
	if err != nil {
		return Booking{}, err
	}
	if room.DepartmentID != command.DepartmentID {
		return Booking{}, ErrForbidden
	}
	fingerprint := staffsupport.Fingerprint(bookingActionCallNext, struct {
		DepartmentID string
		RoomID       string
		ServiceDate  string
		OperationID  string
	}{command.DepartmentID, command.RoomID, command.ServiceDate, command.OperationID})
	var result Booking
	err = m.bookings.WithinQueueTransaction(ctx, func(tx QueueTxStore) error {
		operation, found, findErr := tx.FindBookingOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if operation.OperatorAccountID != operator.AccountID || operation.Action != bookingActionCallNext || operation.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			return json.Unmarshal(operation.ResultData, &result)
		}
		queues, lockErr := tx.ListRoomQueuesForUpdate(ctx, command.RoomID, serviceDate)
		if lockErr != nil {
			return lockErr
		}
		inProgress, lockErr := tx.HasInProgressBookingForUpdate(ctx, command.RoomID, serviceDate)
		if lockErr != nil {
			return lockErr
		}
		if inProgress {
			return ErrRoomQueueBusy
		}
		for _, queue := range queues {
			if queue.CurrentCalledBookingID != "" {
				return ErrRoomQueueBusy
			}
		}
		for _, queue := range queues {
			booking, entry, found, findErr := tx.FindNextQueueBookingForUpdate(ctx, queue, now)
			if findErr != nil {
				return findErr
			}
			if !found {
				continue
			}
			if booking.DepartmentID != command.DepartmentID {
				return ErrForbidden
			}
			calledAt, deadline := now, now.Add(common.CallTimeout)
			booking.Status, booking.CalledAt, booking.CallDeadline = BookingStatusCalled, &calledAt, &deadline
			booking.CallAttempts = entry.CallAttempts + 1
			booking.UpdatedAt, booking.Version = now, booking.Version+1
			entry.CalledAt, entry.CallDeadline = &calledAt, &deadline
			if err := tx.MarkBookingCalled(ctx, booking, queue, entry, booking.Version-1, uuid.NewString()); err != nil {
				return err
			}
			result = booking
			return tx.RecordBookingOperation(ctx, BookingOperationChange{OperationID: command.OperationID, OperatorAccountID: operator.AccountID, BookingID: booking.BookingID, Action: bookingActionCallNext, RequestFingerprint: fingerprint, Result: result})
		}
		return ErrQueueEmpty
	})
	if err != nil {
		return Booking{}, err
	}
	m.invalidateBookingReads(ctx, result.DepartmentID)
	return result, nil
}

func sameBookingDate(left time.Time, right time.Time) bool {
	left, right = left.In(bookingHospitalLocation), right.In(bookingHospitalLocation)
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}
