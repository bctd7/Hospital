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

type CorrectExaminationReportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCorrectExaminationReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CorrectExaminationReportLogic {
	return &CorrectExaminationReportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CorrectExaminationReportLogic) CorrectExaminationReport(req *types.CorrectReportAPIRequest) (resp *types.ExaminationReportResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	display, err := currentDisplaySnapshots(rpcCtx, l.svcCtx, requestID)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.CorrectExaminationReport(rpcCtx, &appointmentv1.CorrectExaminationReportRequest{ReportId: req.ReportID, Content: reportContent(req.ObjectiveFindings, req.Impression, req.Recommendation, req.Notes), CorrectionReason: req.CorrectionReason, ExpectedReportVersion: req.ExpectedReportVersion, OperationId: req.OperationID, RequestId: requestID, ActorDisplayName: display.ActorName})
	if err != nil {
		return nil, err
	}
	return examinationReport(value), nil
}
