package common

import (
	"context"
	"time"
)

// ReportTxStore 描述检查执行和报告版本变更所需的原子写能力。
// 结束实际检查时释放容量并进入报告待完成；后续发布报告只提交报告版本和完成状态。
type ReportTxStore interface {
	FindBookingOperation(ctx context.Context, operationID string) (BookingOperation, bool, error)
	RecordBookingOperation(ctx context.Context, change BookingOperationChange) error
	GetBookingForUpdate(ctx context.Context, bookingID string) (Booking, error)
	StartExamination(ctx context.Context, booking Booking, expectedVersion int64) error
	EndExamination(ctx context.Context, booking Booking, expectedVersion int64) error
	RecordQueueEvent(ctx context.Context, event QueueEvent) error
	GetQueueEntryForUpdate(ctx context.Context, bookingID string) (QueueEntry, error)
	ReleasePatientItemSession(ctx context.Context, bookingID string) error
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
