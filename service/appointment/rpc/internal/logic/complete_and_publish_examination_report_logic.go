package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CompleteAndPublishExaminationReportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCompleteAndPublishExaminationReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CompleteAndPublishExaminationReportLogic {
	return &CompleteAndPublishExaminationReportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CompleteAndPublishExaminationReportLogic) CompleteAndPublishExaminationReport(in *appointmentv1.CompleteAndPublishExaminationReportRequest) (*appointmentv1.ExaminationReport, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.CompleteAndPublishExaminationReport(l.ctx, principal, staffinput.CompleteAndPublishReport{
		BookingID: in.GetBookingId(), Content: reportContent(in.GetContent()),
		ExpectedBookingVersion: in.GetExpectedBookingVersion(), ExpectedReportVersion: in.GetExpectedReportVersion(),
		ActorDisplayName: in.GetActorDisplayName(), DepartmentName: in.GetDepartmentName(), CampusName: in.GetCampusName(),
		Operation: staffinput.Operation{OperationID: in.GetOperationId(), RequestID: in.GetRequestId()},
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return reportResponse(value), nil
}
