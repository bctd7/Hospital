package staff

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

const bookingActionStartExamination = "start_examination"

// StartExamination 在预约项目时间内把待检查预约直接标记为检查中，并记录实际操作者。
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
	fingerprint := staffsupport.Fingerprint(bookingActionStartExamination, struct {
		BookingID        string
		ExpectedVersion  int64
		ActorDisplayName string
		OperationID      string
	}{command.BookingID, command.ExpectedVersion, command.ActorDisplayName, command.OperationID})
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
		if booking.Status != BookingStatusConfirmed {
			return ErrInvalidState
		}
		now := time.Now().In(bookingHospitalLocation)
		if err := validateExaminationStartWindow(booking, now); err != nil {
			return err
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

// validateExaminationStartWindow 使用预约创建时保存的项目窗口快照校验开始时间。
// 起始时间包含、结束时间不包含；配置后续变化不会改写既有预约的时间语义。
func validateExaminationStartWindow(booking Booking, now time.Time) error {
	start, err := bookingSnapshotDateTime(booking.ServiceDate, booking.ItemStartTime)
	if err != nil {
		return err
	}
	end, err := bookingSnapshotDateTime(booking.ServiceDate, booking.ItemEndTime)
	if err != nil || !end.After(start) {
		return fmt.Errorf("%w: invalid stored examination window", ErrInvalidState)
	}
	now = now.In(bookingHospitalLocation)
	if now.Before(start) || !now.Before(end) {
		return fmt.Errorf("%w: examination must start between %s and %s", ErrExaminationWindowClosed, booking.ItemStartTime, booking.ItemEndTime)
	}
	return nil
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
