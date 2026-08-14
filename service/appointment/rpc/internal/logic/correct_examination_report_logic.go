package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CorrectExaminationReportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCorrectExaminationReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CorrectExaminationReportLogic {
	return &CorrectExaminationReportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CorrectExaminationReportLogic) CorrectExaminationReport(in *appointmentv1.CorrectExaminationReportRequest) (*appointmentv1.ExaminationReport, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.CorrectExaminationReport(l.ctx, principal, staffinput.CorrectReport{
		ReportID: in.GetReportId(), Content: reportContent(in.GetContent()), CorrectionReason: in.GetCorrectionReason(),
		ExpectedReportVersion: in.GetExpectedReportVersion(), ActorDisplayName: in.GetActorDisplayName(),
		Operation: staffinput.Operation{OperationID: in.GetOperationId(), RequestID: in.GetRequestId()},
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return reportResponse(value), nil
}
