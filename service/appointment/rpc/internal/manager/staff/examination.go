package staff

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	"hospital/service/appointment/rpc/internal/manager/common"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

const (
	bookingActionStartExamination = "start_examination"
	bookingActionEndExamination   = "end_examination"
)

// StartExamination 只允许当前正在叫号的患者进入检查中。
func (m *Manager) StartExamination(ctx context.Context, operator authn.Principal, command staffinput.StartExamination) (Booking, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return Booking{}, err
	}
	var err error
	command.BookingID, err = staffsupport.NormalizeUUID(command.BookingID, "booking_id")
	if err != nil {
		return Booking{}, err
	}
	command.OperationID, err = staffsupport.NormalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return Booking{}, err
	}
	if command.ExpectedVersion < 1 {
		return Booking{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	command.ActorDisplayName, err = normalizeSnapshotLabel(command.ActorDisplayName, "actor_display_name")
	if err != nil {
		return Booking{}, err
	}
	fingerprint := common.RequestFingerprint(struct {
		BookingID        string
		ExpectedVersion  int64
		ActorDisplayName string
	}{command.BookingID, command.ExpectedVersion, command.ActorDisplayName})
	var result Booking
	err = m.reports.WithinReportTransaction(ctx, func(tx ReportTxStore) error {
		operation, found, findErr := tx.FindBookingOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if operation.OperatorAccountID != operator.AccountID || operation.BookingID != command.BookingID || operation.Action != bookingActionStartExamination || operation.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			return json.Unmarshal(operation.ResultData, &result)
		}
		booking, lockErr := tx.GetBookingForUpdate(ctx, command.BookingID)
		if lockErr != nil {
			return lockErr
		}
		if err := staffsupport.RequireDepartmentScope(operator, booking.DepartmentID); err != nil {
			return err
		}
		if booking.Status != BookingStatusCalled {
			return ErrInvalidState
		}
		now := time.Now().In(bookingHospitalLocation)
		entry, entryErr := tx.GetQueueEntryForUpdate(ctx, booking.BookingID)
		if entryErr != nil {
			return entryErr
		}
		if entry.CallDeadline == nil || !now.Before(*entry.CallDeadline) {
			return ErrCallExpired
		}
		booking.Status = BookingStatusInProgress
		booking.StartedAt = &now
		booking.StartedBy = operator.AccountID
		booking.StartedByDisplayName = command.ActorDisplayName
		booking.UpdatedAt = now
		booking.Version = command.ExpectedVersion + 1
		if err := tx.StartExamination(ctx, booking, command.ExpectedVersion); err != nil {
			return err
		}
		if err := tx.RecordQueueEvent(ctx, common.QueueEvent{EventID: uuid.NewString(), QueueID: entry.QueueID, BookingID: booking.BookingID, EventType: common.QueueEventStarted, OccurredAt: now}); err != nil {
			return err
		}
		result = booking
		return tx.RecordBookingOperation(ctx, BookingOperationChange{
			OperationID: command.OperationID, OperatorAccountID: operator.AccountID,
			BookingID: booking.BookingID, Action: bookingActionStartExamination,
			RequestFingerprint: fingerprint, Result: result,
		})
	})
	if err != nil {
		return Booking{}, err
	}
	m.invalidateBookingReads(ctx, result.DepartmentID)
	return result, nil
}

// EndExamination 结束实际检查、释放预约资源并进入报告待完成。
func (m *Manager) EndExamination(ctx context.Context, operator authn.Principal, command staffinput.EndExamination) (Booking, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return Booking{}, err
	}
	var err error
	command.BookingID, err = staffsupport.NormalizeUUID(command.BookingID, "booking_id")
	if err != nil {
		return Booking{}, err
	}
	command.OperationID, err = staffsupport.NormalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return Booking{}, err
	}
	if command.ExpectedVersion < 1 {
		return Booking{}, fmt.Errorf("%w: expected_version must be positive", ErrInvalid)
	}
	command.ActorDisplayName, err = normalizeSnapshotLabel(command.ActorDisplayName, "actor_display_name")
	if err != nil {
		return Booking{}, err
	}
	fingerprint := common.RequestFingerprint(struct {
		BookingID        string
		ExpectedVersion  int64
		ActorDisplayName string
	}{command.BookingID, command.ExpectedVersion, command.ActorDisplayName})
	var result Booking
	err = m.reports.WithinReportTransaction(ctx, func(tx ReportTxStore) error {
		operation, found, findErr := tx.FindBookingOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if operation.OperatorAccountID != operator.AccountID || operation.BookingID != command.BookingID || operation.Action != bookingActionEndExamination || operation.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			return json.Unmarshal(operation.ResultData, &result)
		}
		booking, lockErr := tx.GetBookingForUpdate(ctx, command.BookingID)
		if lockErr != nil {
			return lockErr
		}
		if err := staffsupport.RequireDepartmentScope(operator, booking.DepartmentID); err != nil {
			return err
		}
		if booking.Status != BookingStatusInProgress {
			return ErrInvalidState
		}
		now := time.Now().In(bookingHospitalLocation)
		entry, entryErr := tx.GetQueueEntryForUpdate(ctx, booking.BookingID)
		if entryErr != nil {
			return entryErr
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
		if err := tx.ReleasePatientItemSession(ctx, booking.BookingID); err != nil {
			return err
		}
		booking.Status = BookingStatusReportPending
		booking.ExaminationEndedAt = &now
		booking.ExaminationEndedBy = operator.AccountID
		booking.ExaminationEndedByDisplayName = command.ActorDisplayName
		booking.UpdatedAt, booking.Version = now, command.ExpectedVersion+1
		if err := tx.EndExamination(ctx, booking, command.ExpectedVersion); err != nil {
			return err
		}
		if err := tx.RecordQueueEvent(ctx, common.QueueEvent{EventID: uuid.NewString(), QueueID: entry.QueueID, BookingID: booking.BookingID, EventType: common.QueueEventEnded, OccurredAt: now}); err != nil {
			return err
		}
		result = booking
		return tx.RecordBookingOperation(ctx, BookingOperationChange{OperationID: command.OperationID, OperatorAccountID: operator.AccountID, BookingID: booking.BookingID, Action: bookingActionEndExamination, RequestFingerprint: fingerprint, Result: result})
	})
	if err != nil {
		return Booking{}, err
	}
	m.invalidateBookingReads(ctx, result.DepartmentID)
	return result, nil
}

func normalizeSnapshotLabel(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len([]rune(value)) > 128 {
		return "", fmt.Errorf("%w: %s is required", ErrInvalid, field)
	}
	return value, nil
}

func normalizeReportContent(content ReportContent, requirePublishable bool) (ReportContent, error) {
	content.ObjectiveFindings = strings.TrimSpace(content.ObjectiveFindings)
	content.Impression = strings.TrimSpace(content.Impression)
	content.Recommendation = strings.TrimSpace(content.Recommendation)
	content.Notes = strings.TrimSpace(content.Notes)
	if requirePublishable && (content.ObjectiveFindings == "" || content.Impression == "") {
		return ReportContent{}, fmt.Errorf("%w: objective_findings and impression are required", ErrInvalid)
	}
	if len([]rune(content.ObjectiveFindings)) > 20000 || len([]rune(content.Impression)) > 10000 ||
		len([]rune(content.Recommendation)) > 5000 || len([]rune(content.Notes)) > 5000 {
		return ReportContent{}, fmt.Errorf("%w: report content is too long", ErrInvalid)
	}
	return content, nil
}
