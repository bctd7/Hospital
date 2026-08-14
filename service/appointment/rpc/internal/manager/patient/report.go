package patient

import (
	"context"

	"hospital/common/authn"
)

// GetMyExaminationReport 只返回当前账号本人预约对应的正式有效报告。
func (m *Manager) GetMyExaminationReport(ctx context.Context, patient authn.Principal, bookingID string) (ExaminationReport, error) {
	if err := requirePatient(patient); err != nil {
		return ExaminationReport{}, err
	}
	bookingID, err := normalizeUUID(bookingID, "booking_id")
	if err != nil {
		return ExaminationReport{}, err
	}
	report, err := m.reports.GetReportByBooking(ctx, bookingID, true)
	if err != nil {
		return ExaminationReport{}, err
	}
	if report.PatientAccountID != patient.AccountID {
		return ExaminationReport{}, ErrNotFound
	}
	return report, nil
}

// ListMyExaminationReports 分页返回当前账号本人已经正式发布的报告。
func (m *Manager) ListMyExaminationReports(ctx context.Context, patient authn.Principal, page, pageSize int64) (Page[ExaminationReport], error) {
	if err := requirePatient(patient); err != nil {
		return Page[ExaminationReport]{}, err
	}
	page, pageSize, offset, err := normalizeBookingPage(page, pageSize)
	if err != nil {
		return Page[ExaminationReport]{}, err
	}
	values, total, err := m.reports.ListReports(ctx, ReportListFilter{PatientAccountID: patient.AccountID, Status: ReportStatusPublished, Offset: offset, Limit: pageSize}, true)
	if err != nil {
		return Page[ExaminationReport]{}, err
	}
	return Page[ExaminationReport]{Items: values, Page: page, PageSize: pageSize, Total: total}, nil
}
