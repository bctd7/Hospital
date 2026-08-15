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
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	staffsupport "hospital/service/appointment/rpc/internal/manager/staff/support"
)

const (
	bookingActionSaveReportDraft = "save_report_draft"
	bookingActionCompleteReport  = "complete_and_publish_report"
	bookingActionCorrectReport   = "correct_report"
)

// GetExaminationReport 返回工作人员科室范围内的报告，草稿也可见。
func (m *Manager) GetExaminationReport(ctx context.Context, operator authn.Principal, bookingID string) (ExaminationReport, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionReportRead); err != nil {
		return ExaminationReport{}, err
	}
	bookingID, err := staffsupport.NormalizeUUID(bookingID, "booking_id")
	if err != nil {
		return ExaminationReport{}, err
	}
	report, err := m.reports.GetReportByBooking(ctx, bookingID, false)
	if err != nil {
		return ExaminationReport{}, err
	}
	if err := staffsupport.RequireDepartmentScope(operator, report.DepartmentID); err != nil {
		return ExaminationReport{}, err
	}
	return report, nil
}

// ListExaminationReports 分页返回工作人员权限范围内已经正式发布的报告。
// 草稿仍从预约详情进入，避免把待填写内容混入科室报告查询入口。
func (m *Manager) ListExaminationReports(ctx context.Context, operator authn.Principal, departmentID, keyword string, page, pageSize int64) (Page[ExaminationReport], error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentRead); err != nil {
		return Page[ExaminationReport]{}, err
	}
	departmentID, err := staffsupport.ScopedDepartment(operator, departmentID)
	if err != nil {
		return Page[ExaminationReport]{}, err
	}
	keyword = strings.TrimSpace(keyword)
	if len([]rune(keyword)) > 128 {
		return Page[ExaminationReport]{}, fmt.Errorf("%w: keyword is too long", ErrInvalid)
	}
	page, pageSize, offset, err := staffsupport.NormalizePage(page, pageSize)
	if err != nil {
		return Page[ExaminationReport]{}, err
	}
	values, total, err := m.reports.ListReports(ctx, ReportListFilter{
		DepartmentID: departmentID,
		Keyword:      keyword,
		Status:       ReportStatusPublished,
		Offset:       offset,
		Limit:        pageSize,
	}, true)
	if err != nil {
		return Page[ExaminationReport]{}, err
	}
	return Page[ExaminationReport]{Items: values, Page: page, PageSize: pageSize, Total: total}, nil
}

// ListExaminationReportVersions 返回正式报告全部版本和仍存在的草稿版本。
func (m *Manager) ListExaminationReportVersions(ctx context.Context, operator authn.Principal, reportID string) ([]ReportVersion, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionReportRead); err != nil {
		return nil, err
	}
	reportID, err := staffsupport.NormalizeUUID(reportID, "report_id")
	if err != nil {
		return nil, err
	}
	report, err := m.reports.GetReportByID(ctx, reportID)
	if err != nil {
		return nil, err
	}
	if err := staffsupport.RequireDepartmentScope(operator, report.DepartmentID); err != nil {
		return nil, err
	}
	return m.reports.ListReportVersions(ctx, reportID)
}

// SaveExaminationReportDraft 创建或更新首版草稿；正式发布后不允许再通过此入口修改。
func (m *Manager) SaveExaminationReportDraft(ctx context.Context, operator authn.Principal, command staffinput.SaveReportDraft) (ExaminationReport, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionReportPublish); err != nil {
		return ExaminationReport{}, err
	}
	var err error
	command.BookingID, err = staffsupport.NormalizeUUID(command.BookingID, "booking_id")
	if err != nil {
		return ExaminationReport{}, err
	}
	command.OperationID, err = staffsupport.NormalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return ExaminationReport{}, err
	}
	command.Content, err = normalizeReportContent(command.Content, false)
	if err != nil {
		return ExaminationReport{}, err
	}
	if command.ExpectedReportVersion < 0 {
		return ExaminationReport{}, fmt.Errorf("%w: expected_report_version cannot be negative", ErrInvalid)
	}
	command.ActorDisplayName, err = normalizeSnapshotLabel(command.ActorDisplayName, "actor_display_name")
	if err != nil {
		return ExaminationReport{}, err
	}
	fingerprint := staffsupport.Fingerprint(bookingActionSaveReportDraft, struct {
		BookingID             string
		Content               ReportContent
		ExpectedReportVersion int64
		ActorDisplayName      string
		OperationID           string
	}{command.BookingID, command.Content, command.ExpectedReportVersion, command.ActorDisplayName, command.OperationID})
	var result ExaminationReport
	err = m.reports.WithinReportTransaction(ctx, func(tx ReportTxStore) error {
		if replayed, replayErr := replayReportOperation(ctx, tx, operator, command.OperationID, command.BookingID, bookingActionSaveReportDraft, fingerprint, &result); replayed || replayErr != nil {
			return replayErr
		}
		booking, lockErr := tx.GetBookingForUpdate(ctx, command.BookingID)
		if lockErr != nil {
			return lockErr
		}
		if err := staffsupport.RequireDepartmentScope(operator, booking.DepartmentID); err != nil {
			return err
		}
		if booking.Status != BookingStatusInProgress && booking.Status != BookingStatusReportPending {
			return ErrInvalidState
		}
		now := time.Now().In(bookingHospitalLocation)
		report, found, findErr := tx.GetReportByBookingForUpdate(ctx, booking.BookingID)
		if findErr != nil {
			return findErr
		}
		if !found {
			if command.ExpectedReportVersion != 0 {
				return ErrVersionConflict
			}
			report, version := newDraftReport(booking, operator.AccountID, command.ActorDisplayName, command.Content, now)
			if err := tx.CreateDraftReport(ctx, report, version); err != nil {
				return err
			}
			report.CurrentVersion = &version
			result = report
		} else {
			if report.Status != ReportStatusDraft || report.CurrentVersion == nil || report.CurrentVersion.Status != ReportVersionStatusDraft {
				return ErrInvalidState
			}
			if report.Version != command.ExpectedReportVersion {
				return ErrVersionConflict
			}
			version := *report.CurrentVersion
			version.ReportContent = command.Content
			version.AuthoredBy = operator.AccountID
			version.AuthoredByDisplayName = command.ActorDisplayName
			version.UpdatedAt = now
			report.Version++
			report.UpdatedAt = now
			if err := tx.UpdateDraftReport(ctx, report, version, command.ExpectedReportVersion); err != nil {
				return err
			}
			report.CurrentVersion = &version
			result = report
		}
		return recordReportOperation(ctx, tx, operator, command.OperationID, booking.BookingID, bookingActionSaveReportDraft, fingerprint, result)
	})
	return result, err
}

// CompleteAndPublishExaminationReport 发布已经结束检查的首版报告，不再重复释放预约资源。
func (m *Manager) CompleteAndPublishExaminationReport(ctx context.Context, operator authn.Principal, command staffinput.CompleteAndPublishReport) (ExaminationReport, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionAppointmentUpdate); err != nil {
		return ExaminationReport{}, err
	}
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionReportPublish); err != nil {
		return ExaminationReport{}, err
	}
	var err error
	command.BookingID, err = staffsupport.NormalizeUUID(command.BookingID, "booking_id")
	if err != nil {
		return ExaminationReport{}, err
	}
	command.OperationID, err = staffsupport.NormalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return ExaminationReport{}, err
	}
	command.Content, err = normalizeReportContent(command.Content, true)
	if err != nil {
		return ExaminationReport{}, err
	}
	if command.ExpectedBookingVersion < 1 || command.ExpectedReportVersion < 0 {
		return ExaminationReport{}, fmt.Errorf("%w: invalid expected version", ErrInvalid)
	}
	command.ActorDisplayName, err = normalizeSnapshotLabel(command.ActorDisplayName, "actor_display_name")
	if err != nil {
		return ExaminationReport{}, err
	}
	command.DepartmentName, err = normalizeSnapshotLabel(command.DepartmentName, "department_name")
	if err != nil {
		return ExaminationReport{}, err
	}
	command.CampusName, err = normalizeSnapshotLabel(command.CampusName, "campus_name")
	if err != nil {
		return ExaminationReport{}, err
	}
	fingerprint := staffsupport.Fingerprint(bookingActionCompleteReport, struct {
		BookingID              string
		Content                ReportContent
		ExpectedBookingVersion int64
		ExpectedReportVersion  int64
		ActorDisplayName       string
		DepartmentName         string
		CampusName             string
		OperationID            string
	}{command.BookingID, command.Content, command.ExpectedBookingVersion, command.ExpectedReportVersion,
		command.ActorDisplayName, command.DepartmentName, command.CampusName, command.OperationID})
	var result ExaminationReport
	err = m.reports.WithinReportTransaction(ctx, func(tx ReportTxStore) error {
		if replayed, replayErr := replayReportOperation(ctx, tx, operator, command.OperationID, command.BookingID, bookingActionCompleteReport, fingerprint, &result); replayed || replayErr != nil {
			return replayErr
		}
		booking, lockErr := tx.GetBookingForUpdate(ctx, command.BookingID)
		if lockErr != nil {
			return lockErr
		}
		if err := staffsupport.RequireDepartmentScope(operator, booking.DepartmentID); err != nil {
			return err
		}
		if booking.Status != BookingStatusReportPending || booking.StartedAt == nil || booking.StartedBy == "" || booking.ExaminationEndedAt == nil {
			return ErrInvalidState
		}
		if booking.Version != command.ExpectedBookingVersion {
			return ErrVersionConflict
		}
		now := time.Now().In(bookingHospitalLocation)
		report, found, findErr := tx.GetReportByBookingForUpdate(ctx, booking.BookingID)
		if findErr != nil {
			return findErr
		}
		if !found {
			if command.ExpectedReportVersion != 0 {
				return ErrVersionConflict
			}
			report, version := newPublishedReport(booking, operator.AccountID, command.ActorDisplayName,
				command.DepartmentName, command.CampusName, command.Content, now)
			if err := tx.CreatePublishedReport(ctx, report, version); err != nil {
				return err
			}
			report.CurrentVersion = &version
			result = report
		} else {
			if report.Status != ReportStatusDraft || report.CurrentVersion == nil || report.CurrentVersion.Status != ReportVersionStatusDraft {
				return ErrInvalidState
			}
			if report.Version != command.ExpectedReportVersion {
				return ErrVersionConflict
			}
			version := *report.CurrentVersion
			version.ReportContent = command.Content
			version.AuthoredBy = operator.AccountID
			version.AuthoredByDisplayName = command.ActorDisplayName
			version.PublishedBy = operator.AccountID
			version.PublishedByDisplayName = command.ActorDisplayName
			version.PublishedAt = &now
			version.Status = ReportVersionStatusPublished
			version.UpdatedAt = now
			report.Status = ReportStatusPublished
			report.PerformedBy = booking.StartedBy
			report.PerformedByDisplayName = booking.StartedByDisplayName
			report.DepartmentName = command.DepartmentName
			report.CampusName = command.CampusName
			report.ExaminationStartedAt = booking.StartedAt
			report.ExaminationCompletedAt = booking.ExaminationEndedAt
			report.CurrentVersionID = version.VersionID
			report.Version++
			report.UpdatedAt = now
			if err := tx.PublishDraftReport(ctx, report, version, command.ExpectedReportVersion); err != nil {
				return err
			}
			report.CurrentVersion = &version
			result = report
		}
		booking.Status = BookingStatusCompleted
		booking.CompletedAt = &now
		booking.CompletedBy = operator.AccountID
		booking.CompletedByDisplayName = command.ActorDisplayName
		booking.UpdatedAt = now
		booking.Version++
		if err := tx.CompleteBooking(ctx, booking, command.ExpectedBookingVersion); err != nil {
			return err
		}
		return recordReportOperation(ctx, tx, operator, command.OperationID, booking.BookingID, bookingActionCompleteReport, fingerprint, result)
	})
	if err == nil {
		m.invalidateBookingReads(ctx, result.DepartmentID)
	}
	return result, err
}

// CorrectExaminationReport 新增正式更正版本，绝不覆盖原始正式版本。
func (m *Manager) CorrectExaminationReport(ctx context.Context, operator authn.Principal, command staffinput.CorrectReport) (ExaminationReport, error) {
	if err := staffsupport.RequirePermission(operator, contractauthz.PermissionReportCorrect); err != nil {
		return ExaminationReport{}, err
	}
	var err error
	command.ReportID, err = staffsupport.NormalizeUUID(command.ReportID, "report_id")
	if err != nil {
		return ExaminationReport{}, err
	}
	command.OperationID, err = staffsupport.NormalizeUUID(command.OperationID, "operation_id")
	if err != nil {
		return ExaminationReport{}, err
	}
	command.Content, err = normalizeReportContent(command.Content, true)
	if err != nil {
		return ExaminationReport{}, err
	}
	command.CorrectionReason = strings.TrimSpace(command.CorrectionReason)
	if command.CorrectionReason == "" || len([]rune(command.CorrectionReason)) > 512 || command.ExpectedReportVersion < 1 {
		return ExaminationReport{}, fmt.Errorf("%w: correction_reason and expected_report_version are required", ErrInvalid)
	}
	command.ActorDisplayName, err = normalizeSnapshotLabel(command.ActorDisplayName, "actor_display_name")
	if err != nil {
		return ExaminationReport{}, err
	}
	fingerprint := staffsupport.Fingerprint(bookingActionCorrectReport, struct {
		ReportID              string
		Content               ReportContent
		CorrectionReason      string
		ExpectedReportVersion int64
		ActorDisplayName      string
		OperationID           string
	}{command.ReportID, command.Content, command.CorrectionReason, command.ExpectedReportVersion, command.ActorDisplayName, command.OperationID})
	var result ExaminationReport
	err = m.reports.WithinReportTransaction(ctx, func(tx ReportTxStore) error {
		operation, found, findErr := tx.FindBookingOperation(ctx, command.OperationID)
		if findErr != nil {
			return findErr
		}
		if found {
			if operation.OperatorAccountID != operator.AccountID || operation.Action != bookingActionCorrectReport || operation.RequestFingerprint != fingerprint {
				return ErrConflict
			}
			return json.Unmarshal(operation.ResultData, &result)
		}
		report, lockErr := tx.GetReportByIDForUpdate(ctx, command.ReportID)
		if lockErr != nil {
			return lockErr
		}
		if err := staffsupport.RequireDepartmentScope(operator, report.DepartmentID); err != nil {
			return err
		}
		if report.Status != ReportStatusPublished || report.CurrentVersion == nil || report.CurrentVersion.Status != ReportVersionStatusPublished {
			return ErrInvalidState
		}
		if report.Version != command.ExpectedReportVersion {
			return ErrVersionConflict
		}
		now := time.Now().In(bookingHospitalLocation)
		version := ReportVersion{
			VersionID: uuid.NewString(), ReportID: report.ReportID,
			VersionNo: report.CurrentVersion.VersionNo + 1, VersionKind: ReportVersionKindCorrection,
			Status: ReportVersionStatusPublished, ReportContent: command.Content,
			CorrectionReason: command.CorrectionReason, AuthoredBy: operator.AccountID,
			AuthoredByDisplayName: command.ActorDisplayName,
			PublishedBy:           operator.AccountID, PublishedByDisplayName: command.ActorDisplayName,
			PublishedAt: &now, CreatedAt: now, UpdatedAt: now,
		}
		previousID := report.CurrentVersion.VersionID
		report.CurrentVersionID = version.VersionID
		report.CurrentVersion = &version
		report.Version++
		report.UpdatedAt = now
		if err := tx.PublishCorrection(ctx, report, previousID, version, command.ExpectedReportVersion); err != nil {
			return err
		}
		result = report
		return recordReportOperation(ctx, tx, operator, command.OperationID, report.BookingID, bookingActionCorrectReport, fingerprint, result)
	})
	return result, err
}

func newReport(booking Booking, now time.Time) ExaminationReport {
	return ExaminationReport{
		ReportID: uuid.NewString(), BookingID: booking.BookingID, PatientAccountID: booking.PatientAccountID,
		PatientDisplayName: booking.PatientDisplayName, PatientPhoneMasked: booking.PatientPhoneMasked,
		DepartmentID: booking.DepartmentID, ItemID: booking.ItemID, ItemName: booking.ItemName,
		RoomID: booking.RoomID, CampusID: booking.CampusID, Building: booking.Building,
		FloorNumber: booking.FloorNumber, RoomNumber: booking.RoomNumber, RoomDisplayName: booking.RoomDisplayName,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
}

func newDraftReport(booking Booking, author, authorDisplayName string, content ReportContent, now time.Time) (ExaminationReport, ReportVersion) {
	report := newReport(booking, now)
	report.Status = ReportStatusDraft
	version := ReportVersion{VersionID: uuid.NewString(), ReportID: report.ReportID, VersionNo: 1, VersionKind: ReportVersionKindInitial, Status: ReportVersionStatusDraft, ReportContent: content, AuthoredBy: author, AuthoredByDisplayName: authorDisplayName, CreatedAt: now, UpdatedAt: now}
	report.CurrentVersionID = version.VersionID
	return report, version
}

func newPublishedReport(booking Booking, author, authorDisplayName, departmentName, campusName string, content ReportContent, now time.Time) (ExaminationReport, ReportVersion) {
	report := newReport(booking, now)
	report.Status = ReportStatusPublished
	report.PerformedBy = booking.StartedBy
	report.PerformedByDisplayName = booking.StartedByDisplayName
	report.DepartmentName = departmentName
	report.CampusName = campusName
	report.ExaminationStartedAt = booking.StartedAt
	report.ExaminationCompletedAt = booking.ExaminationEndedAt
	version := ReportVersion{VersionID: uuid.NewString(), ReportID: report.ReportID, VersionNo: 1, VersionKind: ReportVersionKindInitial, Status: ReportVersionStatusPublished, ReportContent: content, AuthoredBy: author, AuthoredByDisplayName: authorDisplayName, PublishedBy: author, PublishedByDisplayName: authorDisplayName, PublishedAt: &now, CreatedAt: now, UpdatedAt: now}
	report.CurrentVersionID = version.VersionID
	return report, version
}

func replayReportOperation(ctx context.Context, tx ReportTxStore, operator authn.Principal, operationID, bookingID, action, fingerprint string, result *ExaminationReport) (bool, error) {
	operation, found, err := tx.FindBookingOperation(ctx, operationID)
	if err != nil || !found {
		return found, err
	}
	if operation.OperatorAccountID != operator.AccountID || operation.BookingID != bookingID || operation.Action != action || operation.RequestFingerprint != fingerprint {
		return true, ErrConflict
	}
	return true, json.Unmarshal(operation.ResultData, result)
}

func recordReportOperation(ctx context.Context, tx ReportTxStore, operator authn.Principal, operationID, bookingID, action, fingerprint string, result ExaminationReport) error {
	return tx.RecordBookingOperation(ctx, BookingOperationChange{OperationID: operationID, OperatorAccountID: operator.AccountID, BookingID: bookingID, Action: action, RequestFingerprint: fingerprint, Result: result})
}
