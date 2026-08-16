package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
	patientmanager "hospital/service/appointment/rpc/internal/manager/patient"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
)

const reportSelect = `
SELECT r.id, r.booking_id, r.patient_account_id, r.patient_display_name_snapshot,
       r.patient_phone_masked_snapshot, r.department_id, r.department_name_snapshot,
       r.item_id, r.item_name_snapshot, r.room_id, r.campus_id_snapshot, r.campus_name_snapshot,
       r.building_snapshot, r.floor_number_snapshot, r.room_number_snapshot,
       r.status, COALESCE(r.performed_by, ''), COALESCE(r.performed_by_display_name_snapshot, ''), r.examination_started_at,
	       r.examination_completed_at, r.current_version_id,
       r.version, r.created_at, r.updated_at,
       v.id, v.version_no, v.version_kind, v.status,
       v.objective_findings, v.impression, v.recommendation, v.notes,
       COALESCE(v.correction_reason, ''), v.authored_by, v.authored_by_display_name_snapshot,
       COALESCE(v.published_by, ''), COALESCE(v.published_by_display_name_snapshot, ''),
       v.published_at, v.created_at, v.updated_at
FROM appointment_examination_reports r
JOIN appointment_examination_report_versions v ON v.id = r.current_version_id`

func (s *Store) GetReportByBooking(ctx context.Context, bookingID string, publishedOnly bool) (appointmentmanager.ExaminationReport, error) {
	query := reportSelect + " WHERE r.booking_id = ?"
	if publishedOnly {
		query += " AND r.status = 'published' AND v.status = 'published'"
	}
	value, err := scanReport(s.db.QueryRowContext(ctx, query, bookingID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ExaminationReport{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.ExaminationReport{}, fmt.Errorf("get examination report by booking: %w", err)
	}
	return value, nil
}

func (s *Store) GetReportByID(ctx context.Context, reportID string) (appointmentmanager.ExaminationReport, error) {
	value, err := scanReport(s.db.QueryRowContext(ctx, reportSelect+" WHERE r.id = ?", reportID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ExaminationReport{}, appointmentmanager.ErrNotFound
	}
	if err != nil {
		return appointmentmanager.ExaminationReport{}, fmt.Errorf("get examination report: %w", err)
	}
	return value, nil
}

func (s *Store) ListReports(ctx context.Context, filter appointmentmanager.ReportListFilter, publishedOnly bool) ([]appointmentmanager.ExaminationReport, int64, error) {
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 8)
	if filter.PatientAccountID != "" {
		conditions, args = append(conditions, "r.patient_account_id = ?"), append(args, filter.PatientAccountID)
	}
	if filter.DepartmentID != "" {
		conditions, args = append(conditions, "r.department_id = ?"), append(args, filter.DepartmentID)
	}
	if filter.Keyword != "" {
		keyword := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(filter.Keyword, `\`, `\\`), `%`, `\%`), `_`, `\_`)
		conditions = append(conditions, "(r.patient_display_name_snapshot LIKE ? ESCAPE '\\\\' OR RIGHT(r.patient_phone_masked_snapshot, 4) = ? OR r.item_name_snapshot LIKE ? ESCAPE '\\\\')")
		args = append(args, "%"+keyword+"%", filter.Keyword, "%"+keyword+"%")
	}
	if filter.Status != "" {
		conditions, args = append(conditions, "r.status = ?"), append(args, filter.Status)
	}
	if publishedOnly {
		conditions = append(conditions, "r.status = 'published'", "v.status = 'published'")
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int64
	countQuery := `SELECT COUNT(*) FROM appointment_examination_reports r JOIN appointment_examination_report_versions v ON v.id = r.current_version_id` + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count examination reports: %w", err)
	}
	queryArgs := append(args, filter.Limit, filter.Offset)
	rows, err := s.db.QueryContext(ctx, reportSelect+where+" ORDER BY r.updated_at DESC, r.id DESC LIMIT ? OFFSET ?", queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list examination reports: %w", err)
	}
	defer rows.Close()
	values := make([]appointmentmanager.ExaminationReport, 0)
	for rows.Next() {
		value, scanErr := scanReport(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan examination report list: %w", scanErr)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate examination reports: %w", err)
	}
	return values, total, nil
}

func (s *Store) ListReportVersions(ctx context.Context, reportID string) ([]appointmentmanager.ReportVersion, error) {
	rows, err := s.db.QueryContext(ctx, reportVersionSelect+" WHERE report_id = ? ORDER BY version_no DESC", reportID)
	if err != nil {
		return nil, fmt.Errorf("list examination report versions: %w", err)
	}
	defer rows.Close()
	values := make([]appointmentmanager.ReportVersion, 0)
	for rows.Next() {
		value, scanErr := scanReportVersion(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan examination report version: %w", scanErr)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate examination report versions: %w", err)
	}
	return values, nil
}

func (s *Store) WithinReportTransaction(ctx context.Context, fn func(appointmentmanager.ReportTxStore) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin report transaction: %w", err)
	}
	txStore := &reportTxStore{bookingTxStore: &bookingTxStore{tx: tx}}
	if err := fn(txStore); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback report transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit report transaction: %w", err)
	}
	return nil
}

type reportScanner interface{ Scan(dest ...any) error }

func scanReport(scanner reportScanner) (appointmentmanager.ExaminationReport, error) {
	var value appointmentmanager.ExaminationReport
	var version appointmentmanager.ReportVersion
	var startedAt, completedAt, publishedAt sql.NullTime
	err := scanner.Scan(
		&value.ReportID, &value.BookingID, &value.PatientAccountID, &value.PatientDisplayName,
		&value.PatientPhoneMasked, &value.DepartmentID, &value.DepartmentName,
		&value.ItemID, &value.ItemName, &value.RoomID, &value.CampusID, &value.CampusName,
		&value.Building, &value.FloorNumber, &value.RoomNumber,
		&value.Status, &value.PerformedBy, &value.PerformedByDisplayName, &startedAt, &completedAt, &value.CurrentVersionID,
		&value.Version, &value.CreatedAt, &value.UpdatedAt,
		&version.VersionID, &version.VersionNo, &version.VersionKind, &version.Status,
		&version.ObjectiveFindings, &version.Impression, &version.Recommendation, &version.Notes,
		&version.CorrectionReason, &version.AuthoredBy, &version.AuthoredByDisplayName,
		&version.PublishedBy, &version.PublishedByDisplayName, &publishedAt,
		&version.CreatedAt, &version.UpdatedAt,
	)
	if err != nil {
		return appointmentmanager.ExaminationReport{}, err
	}
	version.ReportID = value.ReportID
	if startedAt.Valid {
		value.ExaminationStartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		value.ExaminationCompletedAt = &completedAt.Time
	}
	if publishedAt.Valid {
		version.PublishedAt = &publishedAt.Time
	}
	value.RoomDisplayName = appointmentmanager.FormatRoomDisplayName(value.Building, value.FloorNumber, value.RoomNumber)
	value.CurrentVersion = &version
	return value, nil
}

const reportVersionSelect = `
SELECT id, report_id, version_no, version_kind, status,
       objective_findings, impression, recommendation, notes,
       COALESCE(correction_reason, ''), authored_by, authored_by_display_name_snapshot,
       COALESCE(published_by, ''), COALESCE(published_by_display_name_snapshot, ''),
       published_at, created_at, updated_at
FROM appointment_examination_report_versions`

func scanReportVersion(scanner reportScanner) (appointmentmanager.ReportVersion, error) {
	var value appointmentmanager.ReportVersion
	var publishedAt sql.NullTime
	err := scanner.Scan(&value.VersionID, &value.ReportID, &value.VersionNo, &value.VersionKind, &value.Status,
		&value.ObjectiveFindings, &value.Impression, &value.Recommendation, &value.Notes,
		&value.CorrectionReason, &value.AuthoredBy, &value.AuthoredByDisplayName,
		&value.PublishedBy, &value.PublishedByDisplayName, &publishedAt,
		&value.CreatedAt, &value.UpdatedAt)
	if publishedAt.Valid {
		value.PublishedAt = &publishedAt.Time
	}
	return value, err
}

var (
	_ patientmanager.ReportStore = (*Store)(nil)
	_ staffmanager.ReportStore   = (*Store)(nil)
)
