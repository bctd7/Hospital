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

type GetMyExaminationReportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMyExaminationReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyExaminationReportLogic {
	return &GetMyExaminationReportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMyExaminationReportLogic) GetMyExaminationReport(req *types.BookingPathRequest) (resp *types.ExaminationReportResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.GetMyExaminationReport(rpcCtx, &appointmentv1.GetExaminationReportRequest{BookingId: req.BookingID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return examinationReport(value), nil
}
