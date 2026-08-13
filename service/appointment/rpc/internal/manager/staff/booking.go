package staff

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

const (
	bookingActionCheckIn       = "check_in"
	bookingActionDeleteByStaff = "delete_by_staff"
	bookingDateLayout          = "2006-01-02"
)

var bookingHospitalLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}()

type ListBookingsQuery struct {
	DepartmentID string
	ServiceDate  string
	Session      Session
	ItemID       string
	RoomID       string
	Status       BookingStatus
	Page         int64
	PageSize     int64
}

type CheckInBookingCommand struct {
	BookingID       string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type DeleteBookingCommand struct {
	BookingID   string
	OperationID string
	Reason      string
	RequestID   string
}

func (m *Manager) GetBooking(ctx context.Context, operator authn.Principal, bookingID string) (Booking, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Booking{}, err
	}
	bookingID, err := normalizeUUID(bookingID, "booking_id")
	if err != nil {
		return Booking{}, err
	}
	value, err := m.bookings.GetBooking(ctx, bookingID)
	if err != nil {
		return Booking{}, err
	}
	if err := requireScope(operator, value.DepartmentID); err != nil {
		return Booking{}, err
	}
	return value, nil
}

func (m *Manager) ListBookings(ctx context.Context, operator authn.Principal, query ListBookingsQuery) (Page[Booking], error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Page[Booking]{}, err
	}
	departmentID, err := scopedDepartment(operator, query.DepartmentID)
	if err != nil {
		return Page[Booking]{}, err
	}
	page, pageSize, offset, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[Booking]{}, err
	}
	filter := BookingListFilter{DepartmentID: departmentID, Offset: offset, Limit: pageSize}
	if strings.TrimSpace(query.ServiceDate) != "" {
		date, parseErr := time.ParseInLocation(bookingDateLayout, strings.TrimSpace(query.ServiceDate), bookingHospitalLocation)
		if parseErr != nil {
			return Page[Booking]{}, fmt.Errorf("%w: service_date must use YYYY-MM-DD", ErrInvalid)
		}
		filter.ServiceDate = &date
	}
	if query.Session != "" {
		query.Session = Session(strings.TrimSpace(string(query.Session)))
		if !query.Session.Valid() {
			return Page[Booking]{}, fmt.Errorf("%w: invalid session", ErrInvalid)
		}
		filter.Session = query.Session
	}
	if query.Status != "" {
		query.Status = BookingStatus(strings.TrimSpace(string(query.Status)))
		if !query.Status.Valid() {
			return Page[Booking]{}, fmt.Errorf("%w: invalid booking status", ErrInvalid)
		}
		filter.Status = query.Status
	}
	if strings.TrimSpace(query.ItemID) != "" {
		filter.ItemID, err = normalizeUUID(query.ItemID, "item_id")
		if err != nil {
			return Page[Booking]{}, err
		}
	}
	if strings.TrimSpace(query.RoomID) != "" {
		filter.RoomID, err = normalizeUUID(query.RoomID, "room_id")
		if err != nil {
			return Page[Booking]{}, err
		}
	}
	values, total, err := m.bookings.ListBookings(ctx, filter)
	if err != nil {
		return Page[Booking]{}, err
	}
	return Page[Booking]{Items: values, Page: page, PageSize: pageSize, Total: total}, nil
}

func (m *Manager) CheckInBooking(ctx context.Context, operator authn.Principal, command CheckInBookingCommand) (Booking, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
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
	command.RequestID = strings.TrimSpace(command.RequestID)
	fingerprintValue := fingerprint(bookingActionCheckIn, struct {
		BookingID       string
		ExpectedVersion int64
		OperationID     string
	}{command.BookingID, command.ExpectedVersion, command.OperationID})
	var result Booking
	err = m.bookings.WithinBookingTransaction(ctx, func(tx BookingTxStore) error {
		operation, found, findErr := tx.FindBookingOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if operation.OperatorAccountID != operator.AccountID || operation.BookingID != command.BookingID || operation.Action != bookingActionCheckIn || operation.RequestFingerprint != fingerprintValue {
				return ErrConflict
			}
			return json.Unmarshal(operation.ResultData, &result)
		}
		booking, lockErr := tx.GetBookingForUpdate(ctx, command.BookingID)
		if lockErr != nil {
			return lockErr
		}
		if err := requireScope(operator, booking.DepartmentID); err != nil {
			return err
		}
		if booking.Status != BookingStatusConfirmed {
			return ErrInvalidState
		}
		now := time.Now().In(bookingHospitalLocation)
		if booking.ServiceDate.In(bookingHospitalLocation).Format(bookingDateLayout) != now.Format(bookingDateLayout) {
			return fmt.Errorf("%w: booking can only be checked in on its service date", ErrInvalidState)
		}
		booking.Status = BookingStatusCheckedIn
		booking.CheckedInAt = &now
		booking.CheckedInBy = operator.AccountID
		booking.UpdatedAt = now
		booking.Version = command.ExpectedVersion + 1
		if err := tx.CheckInBooking(ctx, booking, command.ExpectedVersion); err != nil {
			return err
		}
		result = booking
		return tx.RecordBookingOperation(ctx, BookingOperationChange{
			OperationID: command.OperationID, OperatorAccountID: operator.AccountID,
			BookingID: booking.BookingID, Action: bookingActionCheckIn,
			RequestFingerprint: fingerprintValue, Result: result,
		})
	})
	if err != nil {
		return Booking{}, err
	}
	m.invalidateBookingReads(ctx, result.DepartmentID)
	return result, nil
}

func (m *Manager) DeleteBooking(ctx context.Context, operator authn.Principal, command DeleteBookingCommand) (string, error) {
	if err := requirePermission(operator, contractauthz.PermissionAppointmentCancel); err != nil {
		return "", err
	}
	var err error
	command.BookingID, err = normalizeUUID(command.BookingID, "booking_id")
	if err != nil {
		return "", err
	}
	command.OperationID, err = normalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return "", err
	}
	command.Reason = strings.TrimSpace(command.Reason)
	if len(command.Reason) > 256 {
		return "", fmt.Errorf("%w: reason is too long", ErrInvalid)
	}
	command.RequestID = strings.TrimSpace(command.RequestID)
	fingerprintValue := fingerprint(bookingActionDeleteByStaff, struct {
		BookingID   string
		OperationID string
		Reason      string
	}{command.BookingID, command.OperationID, command.Reason})
	var deletedID, departmentID string
	err = m.bookings.WithinBookingTransaction(ctx, func(tx BookingTxStore) error {
		operation, found, findErr := tx.FindBookingOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if operation.OperatorAccountID != operator.AccountID || operation.BookingID != command.BookingID || operation.Action != bookingActionDeleteByStaff || operation.RequestFingerprint != fingerprintValue {
				return ErrConflict
			}
			deletedID = operation.BookingID
			return nil
		}
		booking, lockErr := tx.GetBookingForUpdate(ctx, command.BookingID)
		if lockErr != nil {
			return lockErr
		}
		if err := requireScope(operator, booking.DepartmentID); err != nil {
			return err
		}
		capacity, found, capacityErr := tx.FindDateCapacityForUpdate(ctx, booking.RoomID, booking.ServiceDate, booking.Session)
		if capacityErr != nil {
			return capacityErr
		}
		if !found {
			return fmt.Errorf("%w: booking capacity not found", ErrInvalidState)
		}
		if err := tx.DecreaseOccupiedCapacity(ctx, capacity.CapacityID); err != nil {
			return err
		}
		if err := tx.DeleteBooking(ctx, booking.BookingID); err != nil {
			return err
		}
		deletedID, departmentID = booking.BookingID, booking.DepartmentID
		return tx.RecordBookingOperation(ctx, BookingOperationChange{
			OperationID: command.OperationID, OperatorAccountID: operator.AccountID,
			BookingID: booking.BookingID, Action: bookingActionDeleteByStaff,
			RequestFingerprint: fingerprintValue,
			Result:             map[string]any{"booking_id": booking.BookingID, "deleted": true},
		})
	})
	if err != nil {
		return "", err
	}
	if departmentID != "" {
		m.invalidateBookingReads(ctx, departmentID)
	}
	return deletedID, nil
}

func (m *Manager) invalidateBookingReads(ctx context.Context, departmentID string) {
	if m.cache != nil && departmentID != "" {
		_ = m.cache.BumpDepartment(ctx, departmentID)
	}
}
