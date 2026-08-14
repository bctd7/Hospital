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

type SaveExaminationReportDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSaveExaminationReportDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveExaminationReportDraftLogic {
	return &SaveExaminationReportDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SaveExaminationReportDraftLogic) SaveExaminationReportDraft(req *types.SaveReportDraftAPIRequest) (resp *types.ExaminationReportResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	display, err := currentDisplaySnapshots(rpcCtx, l.svcCtx, requestID)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.SaveExaminationReportDraft(rpcCtx, &appointmentv1.SaveExaminationReportDraftRequest{BookingId: req.BookingID, Content: reportContent(req.ObjectiveFindings, req.Impression, req.Recommendation, req.Notes), ExpectedReportVersion: req.ExpectedReportVersion, OperationId: req.OperationID, RequestId: requestID, ActorDisplayName: display.ActorName})
	if err != nil {
		return nil, err
	}
	return examinationReport(value), nil
}
