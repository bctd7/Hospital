package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetExaminationReportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetExaminationReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationReportLogic {
	return &GetExaminationReportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetExaminationReportLogic) GetExaminationReport(in *appointmentv1.GetExaminationReportRequest) (*appointmentv1.ExaminationReport, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.GetExaminationReport(l.ctx, principal, in.GetBookingId())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return reportResponse(value), nil
}
