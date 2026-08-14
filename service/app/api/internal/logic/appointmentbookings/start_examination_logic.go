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

type StartExaminationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStartExaminationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartExaminationLogic {
	return &StartExaminationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartExaminationLogic) StartExamination(req *types.StartExaminationAPIRequest) (resp *types.BookingResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	display, err := currentDisplaySnapshots(rpcCtx, l.svcCtx, requestID)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.StartExamination(rpcCtx, &appointmentv1.StartExaminationRequest{BookingId: req.BookingID, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID, RequestId: requestID, ActorDisplayName: display.ActorName})
	if err != nil {
		return nil, err
	}
	return bookingWithOrganization(rpcCtx, l.svcCtx, requestID, value), nil
}
