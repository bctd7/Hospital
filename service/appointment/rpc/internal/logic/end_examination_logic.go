package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type EndExaminationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEndExaminationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EndExaminationLogic {
	return &EndExaminationLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *EndExaminationLogic) EndExamination(in *appointmentv1.EndExaminationRequest) (*appointmentv1.Booking, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.EndExamination(l.ctx, principal, staffinput.EndExamination{
		BookingID: in.GetBookingId(), ExpectedVersion: in.GetExpectedVersion(), ActorDisplayName: in.GetActorDisplayName(),
		Operation: staffinput.Operation{OperationID: in.GetOperationId(), RequestID: in.GetRequestId()},
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return bookingResponse(value), nil
}
