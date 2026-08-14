// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentbookings

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CompleteAndPublishExaminationReportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCompleteAndPublishExaminationReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CompleteAndPublishExaminationReportLogic {
	return &CompleteAndPublishExaminationReportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CompleteAndPublishExaminationReportLogic) CompleteAndPublishExaminationReport(req *types.CompleteAndPublishReportAPIRequest) (resp *types.ExaminationReportResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	display, err := currentDisplaySnapshots(rpcCtx, l.svcCtx, requestID)
	if err != nil {
		return nil, err
	}
	bookingValue, err := l.svcCtx.Appointment.GetBooking(rpcCtx, &appointmentv1.GetBookingRequest{BookingId: req.BookingID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	departmentName, campusName, err := reportOrganizationSnapshots(rpcCtx, l.svcCtx, bookingValue, requestID)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.CompleteAndPublishExaminationReport(rpcCtx, &appointmentv1.CompleteAndPublishExaminationReportRequest{BookingId: req.BookingID, Content: reportContent(req.ObjectiveFindings, req.Impression, req.Recommendation, req.Notes), ExpectedBookingVersion: req.ExpectedBookingVersion, ExpectedReportVersion: req.ExpectedReportVersion, OperationId: req.OperationID, RequestId: requestID, ActorDisplayName: display.ActorName, DepartmentName: departmentName, CampusName: campusName})
	if err != nil {
		return nil, err
	}
	return examinationReport(value), nil
}
