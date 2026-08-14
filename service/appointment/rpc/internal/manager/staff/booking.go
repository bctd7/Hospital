package staff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

const (
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

// GetBooking 返回工作人员权限范围内的预约详情。
func (m *Manager) GetBooking(ctx context.Context, operator authn.Principal, bookingID string) (Booking, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Booking{}, err
	}
	bookingID, err := staffsupport.NormalizeUUID(bookingID, "booking_id")
	if err != nil {
		return Booking{}, err
	}
	value, err := m.bookings.GetBooking(ctx, bookingID)
	if err != nil {
		return Booking{}, err
	}
	if err := staffsupport.RequireDepartmentScope(operator, value.DepartmentID); err != nil {
		return Booking{}, err
	}
	return value, nil
}

// ListBookings 按工作人员权限范围和筛选条件分页查询预约。
func (m *Manager) ListBookings(ctx context.Context, operator authn.Principal, query staffinput.ListBookings) (Page[Booking], error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Page[Booking]{}, err
	}
	departmentID, err := staffsupport.ScopedDepartment(operator, query.DepartmentID)
	if err != nil {
		return Page[Booking]{}, err
	}
	page, pageSize, offset, err := staffsupport.NormalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[Booking]{}, err
	}
	filter := BookingListFilter{DepartmentID: departmentID, Offset: offset, Limit: pageSize}
	if query.View == "" {
		query.View = BookingListViewActive
	}
	if !query.View.Valid() {
		return Page[Booking]{}, fmt.Errorf("%w: view must be active or completed", ErrInvalid)
	}
	filter.View = query.View
	filter.PatientKeyword = strings.TrimSpace(query.PatientKeyword)
	if len([]rune(filter.PatientKeyword)) > 128 {
		return Page[Booking]{}, fmt.Errorf("%w: patient_keyword is too long", ErrInvalid)
	}
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
		filter.ItemID, err = staffsupport.NormalizeUUID(query.ItemID, "item_id")
		if err != nil {
			return Page[Booking]{}, err
		}
	}
	if strings.TrimSpace(query.RoomID) != "" {
		filter.RoomID, err = staffsupport.NormalizeUUID(query.RoomID, "room_id")
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

func bookingSnapshotDateTime(serviceDate time.Time, clock string) (time.Time, error) {
	parsed, err := time.Parse("15:04:05", strings.TrimSpace(clock))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: invalid stored examination time", ErrInvalidState)
	}
	serviceDate = serviceDate.In(bookingHospitalLocation)
	return time.Date(
		serviceDate.Year(), serviceDate.Month(), serviceDate.Day(),
		parsed.Hour(), parsed.Minute(), parsed.Second(), 0,
		bookingHospitalLocation,
	), nil
}

// DeleteBooking 由工作人员删除预约，并在同一事务中释放容量和患者时段。
func (m *Manager) DeleteBooking(ctx context.Context, operator authn.Principal, command staffinput.DeleteBooking) (string, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentCancel); err != nil {
		return "", err
	}
	var err error
	command.BookingID, err = staffsupport.NormalizeUUID(command.BookingID, "booking_id")
	if err != nil {
		return "", err
	}
	command.OperationID, err = staffsupport.NormalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return "", err
	}
	command.Reason = strings.TrimSpace(command.Reason)
	if len(command.Reason) > 256 {
		return "", fmt.Errorf("%w: reason is too long", ErrInvalid)
	}
	command.RequestID = strings.TrimSpace(command.RequestID)
	fingerprintValue := staffsupport.Fingerprint(bookingActionDeleteByStaff, struct {
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
		if err := staffsupport.RequireDepartmentScope(operator, booking.DepartmentID); err != nil {
			return err
		}
		if booking.Status != BookingStatusConfirmed {
			return ErrInvalidState
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
		if err := tx.ReleasePatientSession(ctx, booking.BookingID); err != nil {
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
		if booking.Status == BookingStatusInProgress {
			return nil, fmt.Errorf("%w: started booking %s blocks configuration change", ErrInvalidState, booking.BookingID)
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
	started := 0
	confirmed := make([]Booking, 0, len(bookings))
	for _, booking := range bookings {
		switch booking.Status {
		case BookingStatusInProgress:
			started++
		case BookingStatusConfirmed:
			confirmed = append(confirmed, booking)
		}
	}
	if int64(started) > window.ActiveCapacity {
		return fmt.Errorf("%w: started bookings exceed requested capacity", ErrInvalidState)
	}
	excess := capacity.OccupiedCapacity - window.ActiveCapacity
	if excess > int64(len(confirmed)) {
		return fmt.Errorf("%w: started bookings block capacity reduction", ErrInvalidState)
	}
	for index := int64(0); index < excess; index++ {
		if err := deleteBookingForConfiguration(ctx, tx, operatorAccountID, configurationOperationID, confirmed[index]); err != nil {
			return err
		}
	}
	return tx.UpdateDateCapacity(ctx, capacity.CapacityID, window.ActiveCapacity, window.WindowID, window.Version)
}
