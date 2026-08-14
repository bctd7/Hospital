package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
)

type reportTxStore struct{ *bookingTxStore }

func (s *reportTxStore) GetReportByBookingForUpdate(ctx context.Context, bookingID string) (appointmentmanager.ExaminationReport, bool, error) {
	value, err := scanReport(s.tx.QueryRowContext(ctx, reportSelect+" WHERE r.booking_id = ? FOR UPDATE", bookingID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ExaminationReport{}, false, nil
	}
	if err != nil {
		return appointmentmanager.ExaminationReport{}, false, fmt.Errorf("lock examination report by booking: %w", err)
	}
	return value, true, nil
}

func (s *reportTxStore) GetReportByIDForUpdate(ctx context.Context, reportID string) (appointmentmanager.ExaminationReport, error) {
	value, err := scanReport(s.tx.QueryRowContext(ctx, reportSelect+" WHERE r.id = ? FOR UPDATE", reportID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ExaminationReport{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.ExaminationReport{}, fmt.Errorf("lock examination report: %w", err)
	}
	return value, nil
}

func (s *reportTxStore) CreateDraftReport(ctx context.Context, report appointmentmanager.ExaminationReport, version appointmentmanager.ReportVersion) error {
	return s.createReport(ctx, report, version)
}

func (s *reportTxStore) CreatePublishedReport(ctx context.Context, report appointmentmanager.ExaminationReport, version appointmentmanager.ReportVersion) error {
	return s.createReport(ctx, report, version)
}

func (s *reportTxStore) createReport(ctx context.Context, report appointmentmanager.ExaminationReport, version appointmentmanager.ReportVersion) error {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_examination_reports
    (id, booking_id, patient_account_id, patient_display_name_snapshot, patient_phone_masked_snapshot,
     department_id, department_name_snapshot, item_id, item_name_snapshot,
     room_id, campus_id_snapshot, campus_name_snapshot, building_snapshot, floor_number_snapshot, room_number_snapshot,
     status, performed_by, performed_by_display_name_snapshot, examination_started_at, examination_completed_at,
     current_version_id, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULL, ?, ?, ?)`,
		report.ReportID, report.BookingID, report.PatientAccountID, report.PatientDisplayName,
		report.PatientPhoneMasked, report.DepartmentID, report.DepartmentName,
		report.ItemID, report.ItemName, report.RoomID, report.CampusID, report.CampusName, report.Building,
		report.FloorNumber, report.RoomNumber, report.Status, report.PerformedBy, report.PerformedByDisplayName,
		report.ExaminationStartedAt, report.ExaminationCompletedAt,
		report.Version, report.CreatedAt, report.UpdatedAt,
	)
	if err != nil {
		return mapWriteError(err, "create examination report")
	}
	if err := s.insertVersion(ctx, version); err != nil {
		return err
	}
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_examination_reports SET current_version_id = ? WHERE id = ? AND current_version_id IS NULL`, version.VersionID, report.ReportID)
	if err != nil {
		return fmt.Errorf("attach examination report version: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	return nil
}

func (s *reportTxStore) UpdateDraftReport(ctx context.Context, report appointmentmanager.ExaminationReport, version appointmentmanager.ReportVersion, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_examination_report_versions
SET objective_findings = ?, impression = ?, recommendation = ?, notes = ?,
    authored_by = ?, authored_by_display_name_snapshot = ?, updated_at = ?
WHERE id = ? AND report_id = ? AND status = 'draft'`,
		version.ObjectiveFindings, version.Impression, version.Recommendation, version.Notes,
		version.AuthoredBy, version.AuthoredByDisplayName, version.UpdatedAt, version.VersionID, report.ReportID)
	if err != nil {
		return fmt.Errorf("update report draft version: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	return s.updateReportVersion(ctx, report.ReportID, expectedVersion, report.UpdatedAt)
}

func (s *reportTxStore) PublishDraftReport(ctx context.Context, report appointmentmanager.ExaminationReport, version appointmentmanager.ReportVersion, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_examination_report_versions
SET status = 'published', objective_findings = ?, impression = ?, recommendation = ?, notes = ?,
    authored_by = ?, authored_by_display_name_snapshot = ?,
    published_by = ?, published_by_display_name_snapshot = ?, published_at = ?, updated_at = ?
WHERE id = ? AND report_id = ? AND status = 'draft'`,
		version.ObjectiveFindings, version.Impression, version.Recommendation, version.Notes,
		version.AuthoredBy, version.AuthoredByDisplayName,
		version.PublishedBy, version.PublishedByDisplayName, version.PublishedAt, version.UpdatedAt,
		version.VersionID, report.ReportID)
	if err != nil {
		return fmt.Errorf("publish report draft version: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	result, err = s.tx.ExecContext(ctx, `
UPDATE appointment_examination_reports
SET status = 'published', performed_by = ?, performed_by_display_name_snapshot = ?,
    department_name_snapshot = ?, campus_name_snapshot = ?,
    examination_started_at = ?, examination_completed_at = ?,
    current_version_id = ?, version = version + 1, updated_at = ?
WHERE id = ? AND status = 'draft' AND version = ?`,
		report.PerformedBy, report.PerformedByDisplayName, report.DepartmentName, report.CampusName,
		report.ExaminationStartedAt, report.ExaminationCompletedAt,
		version.VersionID, report.UpdatedAt, report.ReportID, expectedVersion)
	if err != nil {
		return fmt.Errorf("publish examination report: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	return nil
}

func (s *reportTxStore) PublishCorrection(ctx context.Context, report appointmentmanager.ExaminationReport, previousVersionID string, version appointmentmanager.ReportVersion, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_examination_report_versions SET status = 'superseded', updated_at = ? WHERE id = ? AND report_id = ? AND status = 'published'`, report.UpdatedAt, previousVersionID, report.ReportID)
	if err != nil {
		return fmt.Errorf("supersede report version: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	if err := s.insertVersion(ctx, version); err != nil {
		return err
	}
	result, err = s.tx.ExecContext(ctx, `UPDATE appointment_examination_reports SET current_version_id = ?, version = version + 1, updated_at = ? WHERE id = ? AND status = 'published' AND version = ?`, version.VersionID, report.UpdatedAt, report.ReportID, expectedVersion)
	if err != nil {
		return fmt.Errorf("publish corrected report: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	return nil
}

func (s *reportTxStore) insertVersion(ctx context.Context, version appointmentmanager.ReportVersion) error {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_examination_report_versions
    (id, report_id, version_no, version_kind, status, objective_findings, impression,
     recommendation, notes, correction_reason, authored_by, authored_by_display_name_snapshot,
     published_by, published_by_display_name_snapshot, published_at,
     created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`,
		version.VersionID, version.ReportID, version.VersionNo, version.VersionKind, version.Status,
		version.ObjectiveFindings, version.Impression, version.Recommendation, version.Notes,
		version.CorrectionReason, version.AuthoredBy, version.AuthoredByDisplayName,
		version.PublishedBy, version.PublishedByDisplayName, version.PublishedAt,
		version.CreatedAt, version.UpdatedAt)
	return mapWriteError(err, "create examination report version")
}

func (s *reportTxStore) updateReportVersion(ctx context.Context, reportID string, expectedVersion int64, updatedAt any) error {
	result, err := s.tx.ExecContext(ctx, `UPDATE appointment_examination_reports SET version = version + 1, updated_at = ? WHERE id = ? AND status = 'draft' AND version = ?`, updatedAt, reportID, expectedVersion)
	if err != nil {
		return fmt.Errorf("update examination report version: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return appointmentmanager.ErrVersionConflict
	}
	return nil
}

var _ appointmentmanager.ReportTxStore = (*reportTxStore)(nil)
