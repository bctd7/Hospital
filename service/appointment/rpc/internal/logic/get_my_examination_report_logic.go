package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyExaminationReportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyExaminationReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyExaminationReportLogic {
	return &GetMyExaminationReportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMyExaminationReportLogic) GetMyExaminationReport(in *appointmentv1.GetExaminationReportRequest) (*appointmentv1.ExaminationReport, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.PatientManager.GetMyExaminationReport(l.ctx, principal, in.GetBookingId())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return reportResponse(value), nil
}
