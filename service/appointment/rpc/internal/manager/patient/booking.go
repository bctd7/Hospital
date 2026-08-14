package patient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/appointment/rpc/internal/manager/common"
)

const (
	bookingActionCreate          = "create"
	bookingActionDeleteByPatient = "delete_by_patient"
	bookingDateLayout            = "2006-01-02"
	bookingOptionsTTL            = 20 * time.Second
	patientWeeklyBookingLimit    = int64(10)
)

var hospitalLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}()

type CreateBookingCommand struct {
	ItemID             string
	RoomID             string
	ServiceDate        string
	Session            Session
	PatientDisplayName string
	PatientPhoneMasked string
	OperationID        string
	RequestID          string
}

type DeleteBookingCommand struct {
	BookingID   string
	OperationID string
	Reason      string
	RequestID   string
}

// ListBookingOptions 返回本周仍可预约且未超过截止时间的房间时间选项。
func (m *Manager) ListBookingOptions(ctx context.Context, patient authn.Principal, itemID string) ([]BookingOption, time.Time, time.Time, error) {
	if err := requirePatient(patient); err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	itemID, err := normalizeUUID(itemID, "item_id")
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	item, err := m.projects.GetItemSummary(ctx, itemID)
	if err != nil || item.Status != StatusActive {
		return nil, time.Time{}, time.Time{}, ErrNotFound
	}
	now := time.Now().In(hospitalLocation)
	today := dateOnly(now)
	weekStart, weekEnd := currentWeek(today)
	key := fmt.Sprintf("department:%s:g:%s:booking-options:item:%s:week:%s", item.DepartmentID, cacheGeneration(ctx, m.cache, item.DepartmentID), itemID, weekStart.Format(bookingDateLayout))
	options, _, err := loadCached(ctx, m.cache, &m.flights, key, bookingOptionsTTL, func() ([]BookingOption, bool, error) {
		values, loadErr := m.bookings.ListBookingOptions(ctx, itemID, today, weekEnd)
		return values, loadErr == nil, loadErr
	})
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	available := options[:0]
	for _, option := range options {
		if option.RemainingCapacity <= 0 {
			continue
		}
		cutoff, parseErr := dateTime(option.ServiceDate, option.BookingCutoffTime)
		if parseErr != nil || !now.Before(cutoff) {
			continue
		}
		available = append(available, option)
	}
	return available, weekStart, weekEnd, nil
}

// CreateBooking 在事务中占用患者时段和日期容量，并创建本人预约。
func (m *Manager) CreateBooking(ctx context.Context, patient authn.Principal, command CreateBookingCommand) (Booking, error) {
	if err := requirePatient(patient); err != nil {
		return Booking{}, err
	}
	command, serviceDate, err := normalizeCreateBooking(command)
	if err != nil {
		return Booking{}, err
	}
	now := time.Now().In(hospitalLocation)
	if err := validateCurrentWeekDate(now, serviceDate); err != nil {
		return Booking{}, err
	}
	fingerprint := bookingFingerprint(bookingActionCreate, struct {
		ItemID             string
		RoomID             string
		ServiceDate        string
		Session            Session
		PatientDisplayName string
		PatientPhoneMasked string
		OperationID        string
	}{command.ItemID, command.RoomID, command.ServiceDate, command.Session, command.PatientDisplayName, command.PatientPhoneMasked, command.OperationID})
	bookingID := uuid.NewString()
	var result Booking
	err = m.bookings.WithinBookingTransaction(ctx, func(tx BookingTxStore) error {
		operation, found, findErr := tx.FindBookingOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if operation.OperatorAccountID != patient.AccountID || operation.Action != bookingActionCreate || operation.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			if err := json.Unmarshal(operation.ResultData, &result); err != nil {
				return fmt.Errorf("decode create booking result: %w", err)
			}
			return nil
		}
		weekStart, _ := currentWeek(dateOnly(serviceDate))
		if err := tx.ConsumePatientWeeklyQuota(ctx, patient.AccountID, weekStart, patientWeeklyBookingLimit, now); err != nil {
			return err
		}
		if err := tx.ClaimPatientSession(ctx, patient.AccountID, serviceDate, command.Session, bookingID); err != nil {
			return err
		}

		selection, lockErr := tx.LockBookingSelection(ctx, command.ItemID, command.RoomID, serviceDate, command.Session)
		if lockErr != nil {
			return lockErr
		}
		cutoff, parseErr := dateTime(serviceDate, selection.BookingCutoffTime)
		if parseErr != nil {
			return parseErr
		}
		if !now.Before(cutoff) {
			return ErrBookingClosed
		}

		capacity, found, capacityErr := tx.FindDateCapacityForUpdate(ctx, command.RoomID, serviceDate, command.Session)
		if capacityErr != nil {
			return capacityErr
		}
		if !found {
			capacity = DateCapacity{
				CapacityID: uuid.NewString(), RoomID: command.RoomID, ServiceDate: serviceDate,
				Session: command.Session, TotalCapacity: selection.ActiveCapacity,
				RoomWindowID: selection.RoomWindowID, RoomWindowVersion: selection.RoomWindowVersion,
				Version: 1, CreatedAt: now, UpdatedAt: now,
			}
			if err := tx.CreateDateCapacity(ctx, capacity); err != nil {
				return err
			}
			capacity, found, capacityErr = tx.FindDateCapacityForUpdate(ctx, command.RoomID, serviceDate, command.Session)
			if capacityErr != nil || !found {
				if capacityErr != nil {
					return capacityErr
				}
				return fmt.Errorf("%w: date capacity was not created", ErrConflict)
			}
		}
		if capacity.RoomWindowID != selection.RoomWindowID || capacity.RoomWindowVersion != selection.RoomWindowVersion || capacity.TotalCapacity != selection.ActiveCapacity {
			return fmt.Errorf("%w: date capacity is stale", ErrConflict)
		}
		if err := tx.IncreaseOccupiedCapacity(ctx, capacity.CapacityID); err != nil {
			return err
		}
		result = Booking{
			BookingID: bookingID, PatientAccountID: patient.AccountID,
			PatientDisplayName: command.PatientDisplayName, PatientPhoneMasked: command.PatientPhoneMasked,
			PatientPhoneLast4: command.PatientPhoneMasked[len(command.PatientPhoneMasked)-4:],
			DepartmentID:      selection.DepartmentID, ItemID: selection.ItemID, ItemName: selection.ItemName,
			RoomID: selection.RoomID, RoomDisplayName: selection.RoomDisplayName, CampusID: selection.CampusID,
			Building: selection.Building, FloorNumber: selection.FloorNumber, RoomNumber: selection.RoomNumber,
			ServiceDate: serviceDate, Session: selection.Session, Status: BookingStatusConfirmed,
			RoomOpenTime: selection.RoomOpenTime, RoomCloseTime: selection.RoomCloseTime,
			ItemStartTime: selection.ItemStartTime, ItemEndTime: selection.ItemEndTime,
			BookingCutoffTime: selection.BookingCutoffTime,
			Version:           1, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.CreateBooking(ctx, result); err != nil {
			return err
		}
		return tx.RecordBookingOperation(ctx, BookingOperationChange{
			OperationID: command.OperationID, OperatorAccountID: patient.AccountID,
			BookingID: result.BookingID, Action: bookingActionCreate,
			RequestFingerprint: fingerprint, Result: result,
		})
	})
	if err != nil {
		return Booking{}, err
	}
	m.invalidateBookingReads(ctx, result.DepartmentID)
	return result, nil
}

// GetMyBooking 只允许患者读取属于自己的预约。
func (m *Manager) GetMyBooking(ctx context.Context, patient authn.Principal, bookingID string) (Booking, error) {
	if err := requirePatient(patient); err != nil {
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
	if value.PatientAccountID != patient.AccountID {
		return Booking{}, ErrNotFound
	}
	return value, nil
}

// ListMyBookings 分页返回当前患者的预约记录。
func (m *Manager) ListMyBookings(ctx context.Context, patient authn.Principal, view BookingListView, page, pageSize int64) (Page[Booking], error) {
	if err := requirePatient(patient); err != nil {
		return Page[Booking]{}, err
	}
	if view == "" {
		view = BookingListViewActive
	}
	if !view.Valid() {
		return Page[Booking]{}, fmt.Errorf("%w: view must be active or completed", ErrInvalid)
	}
	page, pageSize, offset, err := normalizeBookingPage(page, pageSize)
	if err != nil {
		return Page[Booking]{}, err
	}
	values, total, err := m.bookings.ListBookings(ctx, BookingListFilter{
		PatientAccountID: patient.AccountID, View: view, Offset: offset, Limit: pageSize,
	})
	if err != nil {
		return Page[Booking]{}, err
	}
	return Page[Booking]{Items: values, Page: page, PageSize: pageSize, Total: total}, nil
}

// DeleteMyBooking 取消尚未开始的本人预约，并在同一事务中释放容量。
func (m *Manager) DeleteMyBooking(ctx context.Context, patient authn.Principal, command DeleteBookingCommand) (string, error) {
	if err := requirePatient(patient); err != nil {
		return "", err
	}
	command, err := normalizeDeleteBooking(command)
	if err != nil {
		return "", err
	}
	fingerprint := bookingFingerprint(bookingActionDeleteByPatient, struct {
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
			if operation.OperatorAccountID != patient.AccountID || operation.BookingID != command.BookingID || operation.Action != bookingActionDeleteByPatient || operation.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			deletedID = operation.BookingID
			return nil
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
		start, parseErr := dateTime(booking.ServiceDate, booking.ItemStartTime)
		if parseErr != nil {
			return parseErr
		}
		if !time.Now().In(hospitalLocation).Before(start) {
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
			OperationID: command.OperationID, OperatorAccountID: patient.AccountID,
			BookingID: booking.BookingID, Action: bookingActionDeleteByPatient,
			RequestFingerprint: fingerprint, Result: map[string]any{"booking_id": booking.BookingID, "deleted": true},
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

func normalizeCreateBooking(command CreateBookingCommand) (CreateBookingCommand, time.Time, error) {
	var err error
	command.ItemID, err = normalizeUUID(command.ItemID, "item_id")
	if err != nil {
		return CreateBookingCommand{}, time.Time{}, err
	}
	command.RoomID, err = normalizeUUID(command.RoomID, "room_id")
	if err != nil {
		return CreateBookingCommand{}, time.Time{}, err
	}
	command.OperationID, err = normalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return CreateBookingCommand{}, time.Time{}, err
	}
	command.Session = Session(strings.TrimSpace(string(command.Session)))
	if !command.Session.Valid() {
		return CreateBookingCommand{}, time.Time{}, fmt.Errorf("%w: invalid session", ErrInvalid)
	}
	serviceDate, err := time.ParseInLocation(bookingDateLayout, strings.TrimSpace(command.ServiceDate), hospitalLocation)
	if err != nil {
		return CreateBookingCommand{}, time.Time{}, fmt.Errorf("%w: service_date must use YYYY-MM-DD", ErrInvalid)
	}
	command.ServiceDate = serviceDate.Format(bookingDateLayout)
	command.PatientDisplayName = strings.TrimSpace(command.PatientDisplayName)
	command.PatientPhoneMasked = strings.TrimSpace(command.PatientPhoneMasked)
	if command.PatientDisplayName == "" || len([]rune(command.PatientDisplayName)) > 128 {
		return CreateBookingCommand{}, time.Time{}, fmt.Errorf("%w: patient_display_name is required", ErrInvalid)
	}
	if !validMaskedPhone(command.PatientPhoneMasked) {
		return CreateBookingCommand{}, time.Time{}, fmt.Errorf("%w: patient_phone_masked is invalid", ErrInvalid)
	}
	command.RequestID = strings.TrimSpace(command.RequestID)
	return command, serviceDate, nil
}

func validMaskedPhone(value string) bool {
	if len(value) != 11 || value[3:7] != "****" {
		return false
	}
	for _, part := range []string{value[:3], value[7:]} {
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}

func normalizeDeleteBooking(command DeleteBookingCommand) (DeleteBookingCommand, error) {
	var err error
	command.BookingID, err = normalizeUUID(command.BookingID, "booking_id")
	if err != nil {
		return DeleteBookingCommand{}, err
	}
	command.OperationID, err = normalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return DeleteBookingCommand{}, err
	}
	command.Reason = strings.TrimSpace(command.Reason)
	if len(command.Reason) > 256 {
		return DeleteBookingCommand{}, fmt.Errorf("%w: reason is too long", ErrInvalid)
	}
	command.RequestID = strings.TrimSpace(command.RequestID)
	return command, nil
}

func validateCurrentWeekDate(now, serviceDate time.Time) error {
	today := dateOnly(now)
	start, end := currentWeek(today)
	if serviceDate.Before(today) || serviceDate.Before(start) || serviceDate.After(end) {
		return fmt.Errorf("%w: service_date must be in the current week and not in the past", ErrInvalid)
	}
	return nil
}

func currentWeek(date time.Time) (time.Time, time.Time) {
	weekday := (int(date.Weekday()) + 6) % 7
	start := date.AddDate(0, 0, -weekday)
	return start, start.AddDate(0, 0, 6)
}

func dateOnly(value time.Time) time.Time {
	value = value.In(hospitalLocation)
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, hospitalLocation)
}

func dateTime(date time.Time, clock string) (time.Time, error) {
	parsed, err := time.Parse("15:04:05", clock)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: invalid stored booking time", ErrInvalidState)
	}
	date = date.In(hospitalLocation)
	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, hospitalLocation), nil
}

func normalizeBookingPage(page, pageSize int64) (int64, int64, int64, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return 0, 0, 0, fmt.Errorf("%w: invalid page or page_size", ErrInvalid)
	}
	return page, pageSize, (page - 1) * pageSize, nil
}

func bookingFingerprint(action string, payload any) string {
	data, err := json.Marshal(struct {
		Action  string `json:"action"`
		Payload any    `json:"payload"`
	}{Action: action, Payload: payload})
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (m *Manager) invalidateBookingReads(ctx context.Context, departmentID string) {
	if m.cache != nil && departmentID != "" {
		_ = m.cache.BumpDepartment(ctx, departmentID)
	}
}

func cacheGeneration(ctx context.Context, cache common.Cache, departmentID string) string {
	if cache == nil {
		return "0"
	}
	value, err := cache.DepartmentGeneration(ctx, departmentID)
	if err != nil {
		return "0"
	}
	return value
}
