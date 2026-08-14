package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveExaminationReportDraftLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSaveExaminationReportDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveExaminationReportDraftLogic {
	return &SaveExaminationReportDraftLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SaveExaminationReportDraftLogic) SaveExaminationReportDraft(in *appointmentv1.SaveExaminationReportDraftRequest) (*appointmentv1.ExaminationReport, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.SaveExaminationReportDraft(l.ctx, principal, staffinput.SaveReportDraft{
		BookingID: in.GetBookingId(), Content: reportContent(in.GetContent()), ExpectedReportVersion: in.GetExpectedReportVersion(),
		ActorDisplayName: in.GetActorDisplayName(),
		Operation:        staffinput.Operation{OperationID: in.GetOperationId(), RequestID: in.GetRequestId()},
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return reportResponse(value), nil
}
