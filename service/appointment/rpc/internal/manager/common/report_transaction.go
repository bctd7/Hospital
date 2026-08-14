package common

import (
	"context"
	"time"
)

// ReportTxStore 描述检查执行和报告版本变更必须共享的原子写能力。
// 完成检查、释放容量和发布首版报告通过同一事务提交。
type ReportTxStore interface {
	FindBookingOperation(ctx context.Context, operationID string) (BookingOperation, bool, error)
	RecordBookingOperation(ctx context.Context, change BookingOperationChange) error
	GetBookingForUpdate(ctx context.Context, bookingID string) (Booking, error)
	StartExamination(ctx context.Context, booking Booking, expectedVersion int64) error
	ReleasePatientSession(ctx context.Context, bookingID string) error
	CompleteBooking(ctx context.Context, booking Booking, expectedVersion int64) error
	FindDateCapacityForUpdate(ctx context.Context, roomID string, serviceDate time.Time, session Session) (DateCapacity, bool, error)
	DecreaseOccupiedCapacity(ctx context.Context, capacityID string) error
	GetReportByBookingForUpdate(ctx context.Context, bookingID string) (ExaminationReport, bool, error)
	GetReportByIDForUpdate(ctx context.Context, reportID string) (ExaminationReport, error)
	CreateDraftReport(ctx context.Context, report ExaminationReport, version ReportVersion) error
	UpdateDraftReport(ctx context.Context, report ExaminationReport, version ReportVersion, expectedVersion int64) error
	CreatePublishedReport(ctx context.Context, report ExaminationReport, version ReportVersion) error
	PublishDraftReport(ctx context.Context, report ExaminationReport, version ReportVersion, expectedVersion int64) error
	PublishCorrection(ctx context.Context, report ExaminationReport, previousVersionID string, version ReportVersion, expectedVersion int64) error
}
